package firelfyiii

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/sudores/firefly-iii-bank-sync/bank/dto"
	"github.com/sudores/firefly-iii-bank-sync/util"
	"github.com/rs/zerolog/log"
)

type FireflyiiiConnection struct {
	cl                        *http.Client
	PATToken                  string
	FireflyiiiURL             string
	FireflyiiiTransactionChan chan *dto.TransactionDTO
}

func NewFireflyiiiConnection(PAT, FireflyiiiURL string) *FireflyiiiConnection {
	return &FireflyiiiConnection{
		cl:                        &http.Client{Timeout: time.Second * 30},
		PATToken:                  PAT,
		FireflyiiiURL:             FireflyiiiURL + fireflyiiiAPIPath,
		FireflyiiiTransactionChan: make(chan *dto.TransactionDTO),
	}
}

func (f *FireflyiiiConnection) Serve() {
	for trans := range f.FireflyiiiTransactionChan {
		log.Debug().Msg("ffi Transaction received")
		go func(trans *dto.TransactionDTO) {
			if err := f.createTransaction(trans); err != nil {
				log.Warn().Err(err).Msgf("Failed to create transaction with id: %s", trans.Transaction.ID)
			}
			return
		}(trans)
	}
}

func (f *FireflyiiiConnection) createTransaction(trans *dto.TransactionDTO) error {
	if trans.Transaction.Amount == 0 {
		return errors.New("Transactions with zero amount are not accepted")
	}
	if trans.Transaction.Amount < 0 {
		log.Debug().Msg("Creating withdrawal")
		if err := f.createWithdrawal(trans); err != nil {
			return err
		}
		return nil
	}
	log.Debug().Msg("Creating deposit")
	if err := f.createDeposit(trans); err != nil {
		return err
	}
	return nil
}

func (f *FireflyiiiConnection) createWithdrawal(trans *dto.TransactionDTO) error {
	tr := transactionDTOToTransaction(trans)
	// Get corresponding to transaction account
	accountID, err := f.getCorrespondingAccountID(trans.AccountID)
	accountName, err := f.getCorrespondingAccountName(trans.AccountID)
	if err != nil {
		return err
	}
	tr.Transactions[0].SourceID = accountID
	tr.Transactions[0].SourceName = accountName

	body, err := json.Marshal(tr)
	if err != nil {
		return err
	}
	req, err := f.newRequest(http.MethodPost, fireflyiiiTransactionPath, bytes.NewReader(body))
	if err != nil {
		return err
	}

	resp, err := f.cl.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusUnprocessableEntity { // TODO: Try to undertand why transaction is created but wrong body is returned
		return errors.New(fmt.Sprint("Failed to create transaction, status code: ", resp.StatusCode, " ", string(body)))
	}
	return nil
}

func (f *FireflyiiiConnection) createDeposit(trans *dto.TransactionDTO) error {
	tr := transactionDTOToTransaction(trans)
	// Get corresponding to transaction account
	accountID, err := f.getCorrespondingAccountID(trans.AccountID)
	accountName, err := f.getCorrespondingAccountName(trans.AccountID)
	if err != nil {
		return err
	}
	tr.Transactions[0].DestinationID = accountID
	tr.Transactions[0].DestinationName = accountName

	body, err := json.Marshal(tr)
	if err != nil {
		return err
	}
	req, err := f.newRequest(http.MethodPost, fireflyiiiTransactionPath, bytes.NewReader(body))
	if err != nil {
		return err
	}
	resp, err := f.cl.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprint("Failed to create transaction, status code: ", resp.StatusCode, " ", string(body)))
	}
	return nil
}

func (f *FireflyiiiConnection) getCorrespondingAccountName(accountID string) (string, error) {
	accounts, err := f.getAccountList()
	if err != nil {
		return "", err
	}
	for _, v := range accounts.Data {
		config := extractBPFSConfig(v.Attributes.Notes, accountID)
		if len(config) != 0 {
			if strings.Split(config[0], ":")[1] == accountID {
				return v.Attributes.Name, nil
			}
		}
	}
	return "", errors.New("Valid BPFS config not found for any account")
}

func (f *FireflyiiiConnection) getCorrespondingAccountID(accountID string) (string, error) {
	accounts, err := f.getAccountList()
	if err != nil {
		return "", err
	}
	for _, v := range accounts.Data {
		config := extractBPFSConfig(v.Attributes.Notes, accountID)
		if len(config) != 0 {
			if strings.Split(config[0], ":")[1] == accountID {
				return v.Attributes.ID, nil
			}
		}
	}
	return "", errors.New("Valid BPFS config not found for any account")
}

func extractBPFSConfig(text, substring string) []string {
	re := regexp.MustCompile(`bpfs\..*`)
	match := re.FindStringSubmatch(text)
	return match
}

func (f *FireflyiiiConnection) getAccountList() (*accounts, error) {
	req, err := f.newRequest(http.MethodGet, fireflyiiiAccountsPath, nil)
	if err != nil {
		return nil, err
	}
	resp, err := f.cl.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(fmt.Sprintf("Request to URL %s failed with status code %d", f.FireflyiiiURL+fireflyiiiAccountsPath, resp.StatusCode))
	}
	accountList := accounts{}
	if err := util.HttpResponseToStruct(resp, &accountList); err != nil {
		return nil, err
	}
	return &accountList, err

}

func (f *FireflyiiiConnection) newRequest(method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, f.FireflyiiiURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/vnd.api+json")
	req.Header.Add("Authorization", "Bearer "+f.PATToken)
	return req, nil
}

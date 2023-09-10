package firelfyiii

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/sudores/firefly-iii-bank-sync/bank/dto"
	"github.com/sudores/firefly-iii-bank-sync/util"
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

func (f *FireflyiiiConnection) Serve(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			close(f.FireflyiiiTransactionChan)
			log.Trace().Msg("Firefly-iii channel is closed")
		case trans, ok := <-f.FireflyiiiTransactionChan:
			log.Debug().Msg("ffi Transaction received")
			go func(trans *dto.TransactionDTO) {
				if err := f.createTransaction(ctx, trans); err != nil {
					log.Warn().Err(err).Msgf("Failed to create transaction with id: %s", trans.Transaction.ID)
				}
				return
			}(trans)
			log.Debug().Msg("ffi Transaction created")
			if !ok {
				log.Info().Msg("Shutting down firefly-iii connection. Bye!!!")
				return
			}
		}
	}
}

func (f *FireflyiiiConnection) createTransaction(ctx context.Context, trans *dto.TransactionDTO) error {
	if trans.Transaction.Amount == 0 {
		return errors.New("Transactions with zero amount are not accepted")
	}
	if trans.Transaction.Amount < 0 {
		log.Debug().Msg("Creating withdrawal")
		if err := f.createWithdrawal(ctx, trans); err != nil {
			return err
		}
		return nil
	}
	log.Debug().Msg("Creating deposit")
	if err := f.createDeposit(ctx, trans); err != nil {
		return err
	}
	return nil
}

func (f *FireflyiiiConnection) createWithdrawal(ctx context.Context, trans *dto.TransactionDTO) error {
	tr := transactionDTOToTransaction(trans)
	// Get corresponding to transaction account
	accountID, err := f.getCorrespondingAccountID(ctx, trans.AccountID)
	accountName, err := f.getCorrespondingAccountName(ctx, trans.AccountID)
	if err != nil {
		return err
	}
	tr.Transactions[0].SourceID = accountID
	tr.Transactions[0].SourceName = accountName

	body, err := json.Marshal(tr)
	if err != nil {
		return err
	}
	req, err := f.newRequest(ctx, http.MethodPost, fireflyiiiTransactionPath, bytes.NewReader(body))
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

func (f *FireflyiiiConnection) createDeposit(ctx context.Context, trans *dto.TransactionDTO) error {
	tr := transactionDTOToTransaction(trans)
	// Get corresponding to transaction account
	accountID, err := f.getCorrespondingAccountID(ctx, trans.AccountID)
	accountName, err := f.getCorrespondingAccountName(ctx, trans.AccountID)
	if err != nil {
		return err
	}
	tr.Transactions[0].DestinationID = accountID
	tr.Transactions[0].DestinationName = accountName

	body, err := json.Marshal(tr)
	if err != nil {
		return err
	}
	req, err := f.newRequest(ctx, http.MethodPost, fireflyiiiTransactionPath, bytes.NewReader(body))
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

func (f *FireflyiiiConnection) getCorrespondingAccountName(ctx context.Context, accountID string) (string, error) {
	accounts, err := f.getAccountList(ctx)
	if err != nil {
		return "", err
	}
	for _, v := range accounts.Data {
		config := extractFBSConfig(v.Attributes.Notes, accountID)
		if len(config) != 0 {
			if strings.Split(config[0], ":")[1] == accountID {
				return v.Attributes.Name, nil
			}
		}
	}
	return "", ErrFBSConfigNotFound
}

func (f *FireflyiiiConnection) getCorrespondingAccountID(ctx context.Context, accountID string) (string, error) {
	accounts, err := f.getAccountList(ctx)
	if err != nil {
		return "", err
	}
	for _, v := range accounts.Data {
		config := extractFBSConfig(v.Attributes.Notes, accountID)
		if len(config) != 0 {
			if strings.Split(config[0], ":")[1] == accountID {
				return v.Attributes.ID, nil
			}
		}
	}
	return "", ErrFBSConfigNotFound
}

func extractFBSConfig(text, substring string) []string {
	re := regexp.MustCompile(`fbs\..*`)
	match := re.FindStringSubmatch(text)
	return match
}

func (f *FireflyiiiConnection) getAccountList(ctx context.Context) (*accounts, error) {
	req, err := f.newRequest(ctx, http.MethodGet, fireflyiiiAccountsPath, nil)
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

func (f *FireflyiiiConnection) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, f.FireflyiiiURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/vnd.api+json")
	req.Header.Add("Authorization", "Bearer "+f.PATToken)
	return req, nil
}

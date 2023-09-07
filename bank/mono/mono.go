package mono

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"

	"github.com/sudores/firefly-iii-bank-sync/bank/dto"
	"github.com/sudores/firefly-iii-bank-sync/util"
	"github.com/rs/zerolog/log"
)

type MonoConnection struct {
	monoAPIURL   string
	monoAPIToken string

	listenAddr string

	bPFSHost    string
	bPFSURLPath string

	webhookInitNotify chan uint8
	TransactionChan   chan *dto.TransactionDTO
}

func NewMonoConnetion(APIToken, BPFSHost, listenAddr string) *MonoConnection {
	return &MonoConnection{
		monoAPIURL:      monoAPIURL,
		monoAPIToken:    APIToken,
		listenAddr:      listenAddr,
		bPFSHost:        BPFSHost,
		bPFSURLPath:     "/" + getPathSuffix(),
		TransactionChan: make(chan *dto.TransactionDTO, 2),
	}
}

func (m *MonoConnection) Serve() {
	go func() {
		log.Debug().Msg("Setting up webhook")
		if err := m.webhookSetup(); err != nil {
			log.Fatal().Err(err).Msg("Failed to setup webhook")
		}
		log.Debug().Msg("Mono webhook was setup")
	}()

	log.Debug().Msg("Setting up handlers")
	log.Info().Msgf("Your url is %s", m.bPFSHost+m.bPFSURLPath)
	http.HandleFunc(m.bPFSURLPath, func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength == 0 && r.Method == http.MethodGet {
			fmt.Fprint(w, "")
			return
		}
		if r.Method != http.MethodPost {
			log.Warn().Msg("Bad request received")
			http.Error(w, "Bad request POST and GET only are accepted", http.StatusBadRequest)
			return
		}
		wst := WebhookStatementItem{}
		if err := util.HttpRequestToStruct(r, &wst); err != nil {
			http.Error(w, "Failed to unmarshal json", http.StatusBadRequest)
			return
		}
		log.Debug().Msg("Transaction received")
		m.TransactionChan <- wst.ToTransactionDTO()
		fmt.Fprint(w, "Transaction received")

	})
	log.Info().Msg("Mono starting serving")
	log.Fatal().Msg(http.ListenAndServe(m.listenAddr, nil).Error())
}

func processBadRequests(w http.ResponseWriter, r *http.Request) {
	return
}

func (m *MonoConnection) processWebhookStatementItemPost(w http.ResponseWriter, r *http.Request) {
	wst := WebhookStatementItem{}
	if err := util.HttpRequestToStruct(r, &wst); err != nil {
		http.Error(w, "Failed to unmarshal json", http.StatusBadRequest)
		return
	}
	m.TransactionChan <- wst.ToTransactionDTO()
	log.Debug().Msg("Transaction received")
	fmt.Fprint(w, "Transaction received")
}

func (m *MonoConnection) webhookSetup() error {
	m.checkServeStatus()
	webhook, err := json.Marshal(struct {
		WebHookUrl string `json:"webHookUrl"`
	}{WebHookUrl: m.bPFSHost + m.bPFSURLPath})
	cl := &http.Client{Timeout: time.Second * 30}
	req, err := http.NewRequest(http.MethodPost, m.monoAPIURL+monoWebhookAPIPath, bytes.NewReader(webhook))
	if err != nil {
		return err
	}

	req.Header.Add("X-Token", m.monoAPIToken)
	resp, err := cl.Do(req)
	if err != nil {
		return err
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return errors.New(fmt.Sprintf("Failed to setup webhook %d. API respond %s", resp.StatusCode, string(body)))
	}
	return nil
}

func (m *MonoConnection) checkServeStatus() {
	time.Sleep(time.Second * 2)
	cl := &http.Client{Timeout: time.Second * 5}
	for {
		if resp, err := cl.Get(m.bPFSHost + m.bPFSURLPath); err == nil && resp.StatusCode == http.StatusOK {
			return
		}
		log.Debug().Msg("Mono serve is still warming up")
	}

}

func getPathSuffix() string {
	mbPathLength := 32
	var res []byte
	for i := 0; i <= mbPathLength; i++ {
		char := []byte("abcdefghijklmnopqrstuvwxyz1234567890")
		k, err := rand.Int(rand.Reader, big.NewInt(int64(len(char))))
		if err != nil {
			panic(err)
		}
		res = append(res, char[k.Int64()])
	}
	return string(res)
}

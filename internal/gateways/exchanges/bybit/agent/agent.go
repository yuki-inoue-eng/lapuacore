package agent

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/yuki-inoue-eng/lapuacore/domains"
	"github.com/yuki-inoue-eng/lapuacore/domains/deals"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/bybit/dtos"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/bybit/translators"
)

type PrivateAPIAgent struct {
	httpClient        *http.Client
	signer            *signer
	rateLimiter       *rateLimiter
	orderTranslator   *translators.OrderTranslator
	opErrorTranslator *translators.OperationErrorTranslator

	latencyMeasurer *gateways.LatencyMeasurer

	createOrderURL string
	cancelOrderURL string
	amendOrderURL  string
}

func NewPrivateAPIAgent(credential gateways.Credential, measurer *gateways.LatencyMeasurer) *PrivateAPIAgent {
	const httpRequestTimeoutDuration = 3 * time.Second
	return &PrivateAPIAgent{
		httpClient:        &http.Client{Timeout: httpRequestTimeoutDuration},
		signer:            newSigner(credential),
		rateLimiter:       newRateLimiter(),
		orderTranslator:   translators.NewOrderTranslator(),
		opErrorTranslator: translators.NewOperationErrorTranslator(),

		latencyMeasurer: measurer,

		createOrderURL: "https://" + hostName + createOrderEndpoint,
		cancelOrderURL: "https://" + hostName + cancelOrderEndpoint,
		amendOrderURL:  "https://" + hostName + amendOrderEndpoint,
	}
}

func (a *PrivateAPIAgent) SendOrder(symbol *domains.Symbol, order *deals.Order, handler deals.CreateOrderRespHandler) error {
	if handler == nil {
		return fmt.Errorf("handler is required")
	}
	if err := a.rateLimiter.consume(GroupCreateOrder, 1); err != nil {
		return deals.LateLimitError
	}

	orderDto := a.orderTranslator.TranslateToDto(symbol, order)
	orderJson, err := json.Marshal(orderDto)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, a.createOrderURL, bytes.NewReader(orderJson))
	if err != nil {
		return err
	}
	a.signer.signHttpPost(req, orderJson)

	go func() {
		sendAt := time.Now()
		resp, err := a.httpClient.Do(req)
		receivedAt := time.Now()
		oResp := deals.CreateOrderResp{OrderID: order.GetID()}
		if err != nil {
			handler(oResp, fmt.Errorf("failed to do http request: %v", err))
			return
		}
		if resp.StatusCode != http.StatusOK {
			handler(oResp, fmt.Errorf("failed to send create order (http status %v)", resp.StatusCode))
			return
		}
		_, baseBody, err := a.readRespBody(resp)
		if err != nil {
			handler(oResp, fmt.Errorf("failed to read body: %v", err))
			return
		}
		if !baseBody.IsRetCodeOK() {
			oResp.Err = a.opErrorTranslator.Translate(baseBody)
			handler(oResp, nil)
			return
		}
		arrivedAt := time.UnixMilli(baseBody.TimestampMs)

		if a.latencyMeasurer != nil {
			a.latencyMeasurer.RecordLatency("bybit_create_order_outbound_ms", arrivedAt.Sub(sendAt))
			a.latencyMeasurer.RecordLatency("bybit_create_order_return_ms", receivedAt.Sub(arrivedAt))
			a.latencyMeasurer.RecordLatency("bybit_create_order_round_trip_ms", receivedAt.Sub(sendAt))
		}

		oResp.ArrivedAt = &arrivedAt
		oResp.ConfirmedAt = &receivedAt
		handler(oResp, nil)
	}()
	return nil
}

func (a *PrivateAPIAgent) CancelOrder(symbol *domains.Symbol, order *deals.Order, handler deals.CancelOrderRespHandler) error {
	if handler == nil {
		return fmt.Errorf("handler is required")
	}
	if err := a.rateLimiter.consume(GroupCancelOrder, 1); err != nil {
		return deals.LateLimitError
	}

	cancelDto := a.orderTranslator.TranslateToCancelDto(symbol, order.GetID())
	cancelJson, err := json.Marshal(cancelDto)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, a.cancelOrderURL, bytes.NewReader(cancelJson))
	if err != nil {
		return err
	}
	a.signer.signHttpPost(req, cancelJson)

	go func() {
		sendAt := time.Now()
		resp, err := a.httpClient.Do(req)
		receivedAt := time.Now()
		oResp := deals.CancelOrderResp{OrderID: order.GetID()}
		if err != nil {
			handler(oResp, fmt.Errorf("failed to do http request: %v", err))
			return
		}
		if resp.StatusCode != http.StatusOK {
			handler(oResp, fmt.Errorf("failed to send cancel order (http status %v)", resp.StatusCode))
			return
		}
		_, baseBody, err := a.readRespBody(resp)
		if err != nil {
			handler(oResp, fmt.Errorf("failed to read body: %v", err))
			return
		}
		if !baseBody.IsRetCodeOK() {
			oResp.Err = a.opErrorTranslator.Translate(baseBody)
			handler(oResp, nil)
			return
		}
		arrivedAt := time.UnixMilli(baseBody.TimestampMs)

		if a.latencyMeasurer != nil {
			a.latencyMeasurer.RecordLatency("bybit_cancel_order_outbound_ms", arrivedAt.Sub(sendAt))
			a.latencyMeasurer.RecordLatency("bybit_cancel_order_return_ms", receivedAt.Sub(arrivedAt))
			a.latencyMeasurer.RecordLatency("bybit_cancel_order_round_trip_ms", receivedAt.Sub(sendAt))
		}

		handler(oResp, nil)
	}()
	return nil
}

func (a *PrivateAPIAgent) AmendOrder(symbol *domains.Symbol, order *deals.Order, amendDetail deals.AmendDetail, handler deals.AmendOrderRespHandler) error {
	if handler == nil {
		return fmt.Errorf("handler is required")
	}
	if err := a.rateLimiter.consume(GroupAmendOrder, 1); err != nil {
		return deals.LateLimitError
	}

	amendDetailDto := a.orderTranslator.TranslateToAmendDetailDto(symbol, order.GetID(), &amendDetail)
	amendDetailJson, err := json.Marshal(amendDetailDto)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, a.amendOrderURL, bytes.NewReader(amendDetailJson))
	if err != nil {
		return err
	}
	a.signer.signHttpPost(req, amendDetailJson)

	go func() {
		sendAt := time.Now()
		resp, err := a.httpClient.Do(req)
		receivedAt := time.Now()
		oResp := deals.AmendOrderResp{OrderID: order.GetID()}
		if err != nil {
			handler(oResp, fmt.Errorf("failed to do http request: %v", err))
			return
		}
		if resp.StatusCode != http.StatusOK {
			handler(oResp, fmt.Errorf("failed to send amend order (http status %d)", resp.StatusCode))
			return
		}
		_, baseBody, err := a.readRespBody(resp)
		if err != nil {
			handler(oResp, fmt.Errorf("failed to read body: %v", err))
			return
		}
		if !baseBody.IsRetCodeOK() {
			oResp.Err = a.opErrorTranslator.Translate(baseBody)
			handler(oResp, nil)
			return
		}
		arrivedAt := time.UnixMilli(baseBody.TimestampMs)

		if a.latencyMeasurer != nil {
			a.latencyMeasurer.RecordLatency("bybit_amend_order_outbound_ms", arrivedAt.Sub(sendAt))
			a.latencyMeasurer.RecordLatency("bybit_amend_order_return_ms", receivedAt.Sub(arrivedAt))
			a.latencyMeasurer.RecordLatency("bybit_amend_order_round_trip_ms", receivedAt.Sub(sendAt))
		}

		oResp.Detail = &amendDetail
		oResp.ArrivedAt = &arrivedAt
		oResp.ConfirmedAt = &receivedAt
		handler(oResp, nil)
	}()
	return nil
}

func (a *PrivateAPIAgent) SendOrders(_ *domains.Symbol, _ []*deals.Order, _ deals.CreateOrdersRespHandler) error {
	return errors.New("not implemented")
}

func (a *PrivateAPIAgent) CancelOrders(_ *domains.Symbol, _ []*deals.Order, _ deals.CancelOrdersRespHandler) error {
	return errors.New("not implemented")
}

func (a *PrivateAPIAgent) AmendOrders(_ *domains.Symbol, _ map[*deals.Order]deals.AmendDetail, _ deals.AmendOrdersRespHandler) error {
	return errors.New("not implemented")
}

func (a *PrivateAPIAgent) readRespBody(resp *http.Response) ([]byte, *dtos.BaseRespBody, error) {
	defer resp.Body.Close()
	var r *dtos.BaseRespBody
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	err = json.Unmarshal(body, &r)
	return body, r, err
}

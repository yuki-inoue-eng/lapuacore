package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/yuki-inoue-eng/lapuacore/domains"
	"github.com/yuki-inoue-eng/lapuacore/domains/deals"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/coinex/dtos"
	"github.com/yuki-inoue-eng/lapuacore/internal/gateways/exchanges/coinex/translators"
)

type PrivateAPIAgent struct {
	httpClient        *HttpClient
	signer            *signer
	rateLimiter       *rateLimiter
	orderTranslator   *translators.OrderTranslator
	opErrorTranslator *translators.OperationErrorTranslator

	createOrderURL      string
	cancelOrderURL      string
	amendOrderURL       string
	batchCreateOrderURL string
	batchCancelOrderURL string
}

func NewPrivateAPIAgent(credential gateways.Credential) *PrivateAPIAgent {
	return &PrivateAPIAgent{
		httpClient:        NewHttpClient(),
		signer:            newSigner(credential),
		rateLimiter:       newRateLimiter(),
		orderTranslator:   translators.NewOrderTranslator(),
		opErrorTranslator: translators.NewOperationErrorTranslator(),

		createOrderURL:      "https://" + hostName + createOrderEndpoint,
		cancelOrderURL:      "https://" + hostName + cancelOrderEndpoint,
		amendOrderURL:       "https://" + hostName + amendOrderEndpoint,
		batchCreateOrderURL: "https://" + hostName + batchCreateOrderEndpoint,
		batchCancelOrderURL: "https://" + hostName + batchCancelOrderEndpoint,
	}
}

// Start pre-warms HTTP connections in the background.
func (a *PrivateAPIAgent) Start(ctx context.Context) {
	go a.httpClient.Start(ctx)
}

func (a *PrivateAPIAgent) SendOrders(symbol *domains.Symbol, orders []*deals.Order, handler deals.CreateOrdersRespHandler) error {
	if handler == nil {
		return fmt.Errorf("handler is required")
	}
	if err := a.rateLimiter.consume(GroupOrder, len(orders)); err != nil {
		return err
	}
	listDto := a.orderTranslator.TranslateToListDto(symbol, orders)
	ordersJson, err := json.Marshal(listDto)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, a.batchCreateOrderURL, bytes.NewReader(ordersJson))
	if err != nil {
		return err
	}
	a.signer.signHttpPost(req, ordersJson)

	go func() {
		respMap := deals.CreateOrdersRespMap{}
		for i := range orders {
			respMap[orders[i].GetID()] = nil
		}

		resp, err := a.httpClient.Do(req)
		if err != nil {
			handler(respMap, deals.Error(deals.HttpRequestError, fmt.Sprintf("failed to do http request: %v", err)))
			return
		}
		if resp.StatusCode != http.StatusOK {
			handler(respMap, deals.Error(deals.HttpRequestError, fmt.Sprintf("failed to send create orders (http status %v)", resp.StatusCode)))
			return
		}
		rawBody, baseBody, err := a.readOrdersRespBody(resp)
		if err != nil {
			handler(respMap, fmt.Errorf("failed to read body (%s): %v", string(rawBody), err))
			return
		}
		if !baseBody.IsRetCodeOK() {
			handler(respMap, a.opErrorTranslator.Translate(&baseBody.BaseRespBody))
			return
		}

		confirmedAt := time.Now()
		for i := range orders {
			orderID := orders[i].GetID()
			r := baseBody.Data[i]
			oResp := &deals.CreateOrderResp{OrderID: orderID}
			if !r.IsRetCodeOK() {
				oResp.Err = a.opErrorTranslator.Translate(&r.BaseRespBody)
			} else {
				arrivedAt := time.UnixMilli(r.Data.UpdatedAt)
				oResp.PublicID = strconv.FormatInt(r.Data.PublicOrderID, 10)
				oResp.ArrivedAt = &arrivedAt
				oResp.ConfirmedAt = &confirmedAt
			}
			respMap[orderID] = oResp
		}
		handler(respMap, nil)
	}()
	return nil
}

func (a *PrivateAPIAgent) SendOrder(symbol *domains.Symbol, order *deals.Order, handler deals.CreateOrderRespHandler) error {
	if handler == nil {
		return fmt.Errorf("handler is required")
	}
	if err := a.rateLimiter.consume(GroupOrder, 1); err != nil {
		return err
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
		oResp := deals.CreateOrderResp{OrderID: order.GetID()}
		resp, err := a.httpClient.Do(req)
		confirmedAt := time.Now()
		if err != nil {
			handler(oResp, deals.Error(deals.HttpRequestError, fmt.Sprintf("failed to do http request: %v", err)))
			return
		}
		if resp.StatusCode != http.StatusOK {
			handler(oResp, deals.Error(deals.HttpRequestError, fmt.Sprintf("failed to send create order (http status %v)", resp.StatusCode)))
			return
		}
		rawBody, baseBody, err := a.readOrderRespBody(resp)
		if err != nil {
			handler(oResp, fmt.Errorf("failed to read body (%s): %v", string(rawBody), err))
			return
		}
		if !baseBody.IsRetCodeOK() {
			oResp.Err = a.opErrorTranslator.Translate(&baseBody.BaseRespBody)
			handler(oResp, nil)
			return
		}
		arrivedAt := time.UnixMilli(baseBody.Data.UpdatedAt)
		oResp.PublicID = strconv.FormatInt(baseBody.Data.PublicOrderID, 10)
		oResp.ArrivedAt = &arrivedAt
		oResp.ConfirmedAt = &confirmedAt
		handler(oResp, nil)
	}()
	return nil
}

func (a *PrivateAPIAgent) CancelOrders(symbol *domains.Symbol, orders []*deals.Order, handler deals.CancelOrdersRespHandler) error {
	if handler == nil {
		return fmt.Errorf("handler is required")
	}
	if err := a.rateLimiter.consume(GroupCancel, len(orders)); err != nil {
		return err
	}
	var pIDs []int
	for i := range orders {
		pID, err := strconv.Atoi(orders[i].GetPublicID())
		if err != nil {
			return fmt.Errorf("failed to convert public_id: %v", err)
		}
		pIDs = append(pIDs, pID)
	}
	cancelJson, err := json.Marshal(map[string]any{
		"market":      symbol.Name(),
		"market_type": "FUTURES",
		"order_ids":   pIDs,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, a.batchCancelOrderURL, bytes.NewReader(cancelJson))
	if err != nil {
		return err
	}
	a.signer.signHttpPost(req, cancelJson)

	go func() {
		respMap := deals.CancelOrdersRespMap{}
		for i := range orders {
			respMap[orders[i].GetID()] = nil
		}
		resp, err := a.httpClient.Do(req)
		if err != nil {
			handler(respMap, deals.Error(deals.HttpRequestError, fmt.Sprintf("failed to do http request: %v", err)))
			return
		}
		if resp.StatusCode != http.StatusOK {
			handler(respMap, deals.Error(deals.HttpRequestError, fmt.Sprintf("failed to send cancel orders (http status %v)", resp.StatusCode)))
			return
		}
		_, baseBody, err := a.readOrdersRespBody(resp)
		if err != nil {
			handler(respMap, fmt.Errorf("failed to read body: %v", err))
			return
		}
		if !baseBody.IsRetCodeOK() {
			handler(respMap, a.opErrorTranslator.Translate(&baseBody.BaseRespBody))
			return
		}
		for i := range orders {
			orderID := orders[i].GetID()
			r := baseBody.Data[i]
			oResp := &deals.CancelOrderResp{OrderID: orderID}
			if !r.IsRetCodeOK() {
				oResp.Err = a.opErrorTranslator.Translate(&r.BaseRespBody)
			}
			respMap[orderID] = oResp
		}
		handler(respMap, nil)
	}()
	return nil
}

func (a *PrivateAPIAgent) CancelOrder(symbol *domains.Symbol, order *deals.Order, handler deals.CancelOrderRespHandler) error {
	if handler == nil {
		return fmt.Errorf("handler is required")
	}
	if err := a.rateLimiter.consume(GroupCancel, 1); err != nil {
		return err
	}
	pID, err := strconv.Atoi(order.GetPublicID())
	if err != nil {
		return fmt.Errorf("failed to convert public_id: %v", err)
	}
	cancelJson := []byte(fmt.Sprintf(`{
		"market": "%s",
		"market_type": "FUTURES",
		"order_id": %d
	}`, symbol.Name(), pID))

	req, err := http.NewRequest(http.MethodPost, a.cancelOrderURL, bytes.NewReader(cancelJson))
	if err != nil {
		return err
	}
	a.signer.signHttpPost(req, cancelJson)

	go func() {
		oResp := deals.CancelOrderResp{OrderID: order.GetID()}
		resp, err := a.httpClient.Do(req)
		if err != nil {
			handler(oResp, deals.Error(deals.HttpRequestError, fmt.Sprintf("failed to do http request: %v", err)))
			return
		}
		if resp.StatusCode != http.StatusOK {
			handler(oResp, deals.Error(deals.HttpRequestError, fmt.Sprintf("failed to send cancel order (http status %v)", resp.StatusCode)))
			return
		}
		_, baseBody, err := a.readOrderRespBody(resp)
		if err != nil {
			handler(oResp, fmt.Errorf("failed to read body: %v", err))
			return
		}
		if !baseBody.IsRetCodeOK() {
			oResp.Err = a.opErrorTranslator.Translate(&baseBody.BaseRespBody)
			handler(oResp, nil)
			return
		}
		handler(oResp, nil)
	}()
	return nil
}

func (a *PrivateAPIAgent) AmendOrder(symbol *domains.Symbol, order *deals.Order, amendDetail deals.AmendDetail, handler deals.AmendOrderRespHandler) error {
	if handler == nil {
		return fmt.Errorf("handler is required")
	}
	if err := a.rateLimiter.consume(GroupOrder, 1); err != nil {
		return err
	}
	amendDetailDto, err := a.orderTranslator.TranslateToAmendDetailDto(symbol, order.GetPublicID(), &amendDetail)
	if err != nil {
		return err
	}
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
		oResp := deals.AmendOrderResp{OrderID: order.GetID()}
		resp, err := a.httpClient.Do(req)
		confirmedAt := time.Now()
		if err != nil {
			handler(oResp, deals.Error(deals.HttpRequestError, fmt.Sprintf("failed to do http request: %v", err)))
			return
		}
		if resp.StatusCode != http.StatusOK {
			handler(oResp, deals.Error(deals.HttpRequestError, fmt.Sprintf("failed to send amend order (http status %v)", resp.StatusCode)))
			return
		}
		_, baseBody, err := a.readOrderRespBody(resp)
		if err != nil {
			handler(oResp, fmt.Errorf("failed to read body: %v", err))
			return
		}
		if !baseBody.IsRetCodeOK() {
			oResp.Err = a.opErrorTranslator.Translate(&baseBody.BaseRespBody)
			handler(oResp, nil)
			return
		}
		arrivedAt := time.UnixMilli(baseBody.Data.UpdatedAt)
		oResp.Detail = &amendDetail
		oResp.ArrivedAt = &arrivedAt
		oResp.ConfirmedAt = &confirmedAt
		handler(oResp, nil)
	}()
	return nil
}

func (a *PrivateAPIAgent) AmendOrders(_ *domains.Symbol, _ map[*deals.Order]deals.AmendDetail, _ deals.AmendOrdersRespHandler) error {
	return errors.New("not implemented")
}

func (a *PrivateAPIAgent) readOrdersRespBody(resp *http.Response) ([]byte, *dtos.OrdersRespBody, error) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	var r dtos.OrdersRespBody
	err = json.Unmarshal(body, &r)
	return body, &r, err
}

func (a *PrivateAPIAgent) readOrderRespBody(resp *http.Response) ([]byte, *dtos.OrderRespBody, error) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	var r dtos.OrderRespBody
	err = json.Unmarshal(body, &r)
	return body, &r, err
}

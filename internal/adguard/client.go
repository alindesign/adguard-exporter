package adguard

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/netip"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/alindesign/adguard-exporter/internal/config"
	"github.com/mitchellh/mapstructure"
)

type Client struct {
	conf config.Client
}

func NewClient(conf config.Client) *Client {
	return &Client{conf}
}

func (c *Client) do(ctx context.Context, out any, method string, path string, query url.Values) error {
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", c.conf.Username, c.conf.Password)))
	addr, err := netip.ParseAddrPort(c.Url())
	if err != nil {
		return err
	}

	headers := http.Header{}
	headers.Add("Authorization", fmt.Sprintf("Basic %s", auth))

	endpoint := &url.URL{
		Scheme:   "http",
		Host:     addr.String(),
		Path:     path,
		RawQuery: query.Encode(),
	}

	req := &http.Request{Method: method, URL: endpoint, Header: headers}
	req = req.WithContext(ctx)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d: %v", resp.StatusCode, err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, out); err != nil {
		return err
	}

	return nil
}

func (c *Client) doPost(ctx context.Context, path string, body any, out any) error {
	auth := base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", c.conf.Username, c.conf.Password)))
	url, err := url.Parse(fmt.Sprintf("%s%s", c.conf.Url, path))
	if err != nil {
		return err
	}

	b, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url.String(), bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", fmt.Sprintf("Basic %s", auth))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	if out != nil {
		if err := json.Unmarshal(bodyBytes, out); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) GetStats(ctx context.Context) (*Stats, error) {
	out := &Stats{}
	err := c.do(ctx, out, http.MethodGet, "/control/stats", url.Values{})
	return out, err
}

func (c *Client) GetStatus(ctx context.Context) (*Status, error) {
	out := &Status{}
	err := c.do(ctx, out, http.MethodGet, "/control/status", url.Values{})
	return out, err
}

func (c *Client) GetDhcp(ctx context.Context) (*DhcpStatus, error) {
	out := &DhcpStatus{}
	if err := c.do(ctx, out, http.MethodGet, "/control/dhcp/status", url.Values{}); err != nil {
		return nil, err
	}

	for i := range out.DynamicLeases {
		l := out.DynamicLeases[i]
		l.Type = "dynamic"
		out.DynamicLeases[i] = l
	}

	for i := range out.StaticLeases {
		l := out.StaticLeases[i]
		l.Type = "static"
		out.StaticLeases[i] = l
	}

	out.Leases = slices.Concat(out.DynamicLeases, out.StaticLeases)

	return out, nil
}

// func (c *Client) GetQueryLog(ctx context.Context) (map[string]map[string]int, []QueryTime, QueryPerClient, error) {
func (c *Client) GetQueryLog(ctx context.Context) (map[string]map[string]int, []QueryTime, []logEntry, error) {
	logger := &queryLog{}
	err := c.do(ctx, logger, http.MethodGet, "/control/querylog", url.Values{
		"limit":           {"1000"},
		"response_status": {"all"},
	})
	if err != nil {
		return nil, nil, nil, err
	}

	types, err := c.getQueryTypes(logger)
	if err != nil {
		return nil, nil, nil, err
	}

	times, err := c.getQueryTimes(logger)
	if err != nil {
		return nil, nil, nil, err
	}

	return types, times, log.Log, nil
}

func (c *Client) getQueryTypes(logger *queryLog) (map[string]map[string]int, error) {
	out := map[string]map[string]int{}
	for _, d := range logger.Log {
		if d.Answer != nil && len(d.Answer) > 0 {
			if _, ok := out[d.Client]; !ok {
				out[d.Client] = map[string]int{}
			}
			for i := range d.Answer {
				switch v := d.Answer[i].Value.(type) {
				case string:
					out[d.Client][d.Answer[i].Type]++
				case map[string]any:
					dns65 := &type65{}
					err := mapstructure.Decode(v, dns65)
					if err != nil {
						log.Printf("Warn - could not decode dns65: %v\n", err)
						continue
					}
					out[d.Client]["TYPE"+strconv.Itoa(dns65.Hdr.Rrtype)]++
				}
			}
		}
	}

	return out, nil
}

func (c *Client) getQueryTimes(l *queryLog) ([]QueryTime, error) {
	var out []QueryTime
	for _, q := range l.Log {
		if q.Upstream == "" {
			q.Upstream = "self"
		}

		ms, err := strconv.ParseFloat(q.Elapsed, 32)
		if err != nil {
			log.Printf("ERROR - could not parse query elapsed time %v as float\n", q.Elapsed)
			continue
		}

		out = append(out, QueryTime{
			Elapsed:  time.Millisecond * time.Duration(ms),
			Client:   q.Client,
			Upstream: q.Upstream,
		})
	}
	return out, nil
}

func (c *Client) Url() string {
	return c.conf.Address
}

func (c *Client) SearchClients(ctx context.Context, topClients []map[string]int) (map[string]string, error) {
	reqBody := clientsSearchRequest{}
    for _, c := range topClients {
        for key := range c {
            reqBody.Clients = append(reqBody.Clients, struct {
                ID string `json:"id"`
            }{ID: key})
        }
    }

	var resp []map[string]clientInfo
	err := c.doPost(ctx, "/control/clients/search", reqBody, &resp)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for _, obj := range resp {
		for _, info := range obj {
			for _, id := range info.IDs {
				result[id] = info.Name
			}
		}
	}

	return result, nil
}

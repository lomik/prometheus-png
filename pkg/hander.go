package pkg

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/go-graphite/carbonapi/date"
	"github.com/go-graphite/carbonapi/expr/functions/cairo/png"
	"github.com/go-graphite/carbonapi/expr/types"
	pb "github.com/go-graphite/protocol/carbonapi_v3_pb"
)

var timeNow = time.Now

var exprRegexp = regexp.MustCompile("^g([0-9]+)[.]expr$")

type Handler struct {
	defaultTimeZone *time.Location
	promAddr        string
	queryRangePath  string
	defaultTimeout  time.Duration
}

func NewPNG(promAddr string, queryRangePath string, defaultTimeout time.Duration) *Handler {
	return &Handler{
		defaultTimeZone: time.Local,
		promAddr:        promAddr,
		queryRangePath:  queryRangePath,
		defaultTimeout:  defaultTimeout,
	}
}

func formatLegend(nameMap map[string]string, tpl *template.Template) string {
	if tpl != nil {
		var b bytes.Buffer
		err := tpl.Execute(&b, nameMap)
		if err != nil {
			return err.Error()
		}
		return b.String()
	}
	kv := make([]string, 0, len(nameMap))
	for k, v := range nameMap {
		if k == "__name__" {
			continue
		}
		kv = append(kv, fmt.Sprintf("%s=\"%s\"", k, v))
	}
	if len(kv) > 0 {
		sort.Strings(kv)
		return fmt.Sprintf("%s{%s}", nameMap["__name__"], strings.Join(kv, ","))
	}
	if nameMap["__name__"] != "" {
		return nameMap["__name__"]
	}

	return "{}"
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	type G struct {
		Expr     string
		Legend   string
		Template *template.Template
	}
	params := struct {
		G       map[int]*G    `form:"-"`
		Query   string        `form:"query"`
		From    string        `form:"from"`
		Until   string        `form:"until"`
		TZ      string        `form:"tz"`
		Timeout time.Duration `form:"timeout"`
	}{
		Timeout: h.defaultTimeout,
		G:       map[int]*G{},
	}

	if !parseGetRequest(w, r, &params) {
		return
	}

	if params.Query != "" {
		params.G[0] = &G{Expr: params.Query}
	}

	for k, v := range r.URL.Query() {
		t := exprRegexp.FindStringSubmatch(k)
		if len(t) > 0 {
			graphID, err := strconv.Atoi(t[1])
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			params.G[graphID] = &G{Expr: v[0]}
		}
	}

	if len(params.G) < 1 {
		http.Error(w, "g0.expr required", http.StatusBadRequest)
		return
	}

	for index, gr := range params.G {
		gr.Legend = r.URL.Query().Get(fmt.Sprintf("g%d.legend", index))
		if gr.Legend != "" {
			t, err := template.New("legend").Parse(gr.Legend)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			gr.Template = t
		}
	}

	draftPictureParams := png.GetPictureParams(r, nil)

	ctx, cancel := context.WithTimeout(r.Context(), params.Timeout)
	defer cancel()

	from32 := date.DateParamToEpoch(params.From, params.TZ, timeNow().Add(-24*time.Hour).Unix(), h.defaultTimeZone)
	until32 := date.DateParamToEpoch(params.Until, params.TZ, timeNow().Unix(), h.defaultTimeZone)

	step := (until32 - from32) / int64(2*draftPictureParams.Width)
	if step < 1 {
		step = 1
	}

	metricData := make([]*types.MetricData, 0)

	u, err := url.Parse(h.promAddr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	u.Path = h.queryRangePath

	indexes := make([]int, 0, len(params.G))
	for index, _ := range params.G {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)

	for _, index := range indexes {
		graphData := params.G[index]
		q := u.Query()
		q.Set("query", graphData.Expr)
		q.Set("start", strconv.Itoa(int(from32)))
		q.Set("end", strconv.Itoa(int(until32)))
		q.Set("step", strconv.Itoa(int(step)))
		u.RawQuery = q.Encode()

		req, err := http.NewRequest("GET", u.String(), nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		res, err := http.DefaultClient.Do(req.WithContext(ctx))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		if res.StatusCode != 200 {
			http.Error(w, fmt.Sprintf("prometheus status: %s", res.Status), http.StatusBadGateway)
			return
		}

		promBody, err := ioutil.ReadAll(res.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		promRes := &PrometheusResponse{}
		err = json.Unmarshal(promBody, promRes)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		for _, r := range promRes.Data.Result {
			if len(r.Values) < 1 {
				continue
			}
			step := int64(1)
			if len(r.Values) > 1 {
				step = r.Values[1].Timestamp - r.Values[0].Timestamp
			}
			md := &types.MetricData{
				FetchResponse: pb.FetchResponse{
					Name:              formatLegend(r.Metric, graphData.Template),
					StartTime:         r.Values[0].Timestamp,
					StopTime:          r.Values[len(r.Values)-1].Timestamp,
					StepTime:          step,
					Values:            make([]float64, len(r.Values)),
					ConsolidationFunc: "average",
				},
				ValuesPerPoint: 1,
			}
			for i, v := range r.Values {
				md.FetchResponse.Values[i] = v.Value
			}
			metricData = append(metricData, md)
		}
	}

	pictureParams := png.GetPictureParams(r, metricData)

	response := png.MarshalPNG(pictureParams, metricData)

	w.Header().Set("Content-Type", "image/png")
	w.Write(response)
}

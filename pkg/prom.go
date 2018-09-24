package pkg

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type TimestampValue struct {
	Timestamp int64
	Value     float64
}

type MetricValues struct {
	Metric map[string]string `json:"metric"`
	Values []TimestampValue  `json:"values"`
}

type PrometheusResponseData struct {
	ResultType string         `json:"resultType"`
	Result     []MetricValues `json:"result"`
}

type PrometheusResponse struct {
	Status string                 `json:"status"`
	Data   PrometheusResponseData `json:"data"`
}

func (tv *TimestampValue) UnmarshalJSON(data []byte) error {

	var s []json.RawMessage
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if len(s) != 2 {
		return fmt.Errorf("wrong TimestampValue %#v", string(data))
	}

	var t float64
	var sv string
	if err := json.Unmarshal(s[0], &t); err != nil {
		return err
	}
	if err := json.Unmarshal(s[1], &sv); err != nil {
		return err
	}

	v, err := strconv.ParseFloat(sv, 64)
	if err != nil {
		return err
	}

	tv.Timestamp = int64(t)
	tv.Value = v
	return nil
}

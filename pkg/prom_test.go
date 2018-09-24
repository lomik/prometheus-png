package pkg

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshal(t *testing.T) {
	assert := assert.New(t)

	data, err := ioutil.ReadFile("test1.json")
	if err != nil {
		t.Fatal(err)
	}

	res := &PrometheusResponse{}
	err = json.Unmarshal(data, res)
	assert.NoError(err)

	assert.Len(res.Data.Result[0].Values, 361)
	assert.Equal(int64(1537555404), res.Data.Result[0].Values[360].Timestamp)
	assert.Equal(66039.0, res.Data.Result[0].Values[360].Value)
}

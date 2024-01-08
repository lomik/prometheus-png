package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"regexp"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/go-graphite/carbonapi/expr/functions/cairo/png"
	"github.com/lomik/prometheus-png/pkg"
)

var defaultPictureParams = map[string]interface{}{
	"colorList":          "7EB26D,EAB839,6ED0E0,EF843C,E24D42,1F78C1,BA43A9,705DA0,508642,CCA300,447EBC,C15C17,890F02,0A437C,6D1F62,584477,B7DBAB,F4D598,70DBED,F9BA8F,F29191,82B5D8,E5A8E2,AEA2E0,629E51,E5AC0E,64B0C8,E0752D,BF1B00,0A50A1,962D82,614D93,9AC48A,F2C96D,65C5DB,F9934E,EA6460,5195CE,D683CE,806EB7,3F6833,967302,2F575E,99440A,58140C,052B51,511749,3F2B5B,E0F9D7,FCEACA,CFFAFF,F9E2D2,FCE2DE,BADFF4,F9D9F9,DEDAF7",
	"areaMode":           "all",
	"majorGridLineColor": "666666",
	"minorGridLineColor": "666666",
	"bgcolor":            "171819",
	"areaAlpha":          "0.09",
	"fontName":           "Roboto",
}

type MainConfig struct {
	Listen         string        `toml:"listen"`
	PrometheusAddr string        `toml:"prometheus-addr"`
	PrometheusPath string        `toml:"prometheus-path"`
	PrometheusAuth string		 `toml:"prometheus-auth"`
	CACert		   string		 `toml:"cacert"`
	TimeoutRaw     string        `toml:"timeout"`
	Timeout        time.Duration `toml:"-"`
}

type Config struct {
	Main     MainConfig                          `toml:"main"`
	Template map[string](map[string]interface{}) `toml:"template"`
}

func main() {
	config := Config{
		Main: MainConfig{
			PrometheusAddr: "http://127.0.0.1:9090",
			PrometheusPath: "/api/v1/query_range",
			PrometheusAuth: "",
			Listen:         ":8080",
			CACert:			"",
			Timeout:        10 * time.Second,
			TimeoutRaw:     "10s",
		},
	}
	configFilename := flag.String("config", "", "Config filename. Only TOML format is supported")
	prom := flag.String("prometheus", config.Main.PrometheusAddr, "Prometheus addr")
	promPath := flag.String("prometheus.path", config.Main.PrometheusPath, "Path to query_range endpoint")
	promAuth := flag.String("prometheus.auth", config.Main.PrometheusAuth, "Authorization header")
	caCert := flag.String("cacert", config.Main.CACert, "CA certificate path")
	listen := flag.String("listen", config.Main.Listen, "Listen addr")
	defaultTimeout := flag.Duration("timeout", config.Main.Timeout, "Default timeout for queries")
	configPrintDefault := flag.Bool("config-print-default", false, "Print default config")

	flag.Parse()
	flagset := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { flagset[f.Name] = true })

	if *configPrintDefault {
		enc := toml.NewEncoder(os.Stdout)
		enc.Indent = ""
		if err := enc.Encode(config); err != nil {
			log.Fatal(err)
		}

		fmt.Fprintf(os.Stdout, "\n[template.default]\n")

		if err := enc.Encode(defaultPictureParams); err != nil {
			log.Fatal(err)
		}
		return
	}

	if *configFilename != "" {
		configBodyBytes, err := ioutil.ReadFile(*configFilename)
		if err != nil {
			log.Fatal(err)
		}

		configBody := string(configBodyBytes)

		tmplRe := regexp.MustCompile(`\$\{ENV:([a-zA-Z0-9_]+)\}`)
		configBody = tmplRe.ReplaceAllStringFunc(configBody, func(m string) string {
			parts := tmplRe.FindStringSubmatch(m)
			return os.Getenv(parts[1])
		})

		if _, err := toml.Decode(configBody, &config); err != nil {
			log.Fatal(err)
		}
	}

	if flagset["prometheus"] {
		config.Main.PrometheusAddr = *prom
	}
	if flagset["prometheus.path"] {
		config.Main.PrometheusPath = *promPath
	}
	if flagset["prometheus.auth"] {
		config.Main.PrometheusAuth = *promAuth
	}
	if flagset["cacert"] {
		config.Main.CACert = *caCert
	}
	if flagset["listen"] {
		config.Main.Listen = *listen
	}
	if flagset["timeout"] {
		config.Main.Timeout = *defaultTimeout
	}

	if config.Template == nil {
		config.Template = make(map[string](map[string]interface{}))
	}
	if p, exists := config.Template["default"]; exists {
		for k, v := range defaultPictureParams {
			if _, keyExists := p[k]; !keyExists {
				p[k] = v
			}
		}
	} else {
		config.Template["default"] = defaultPictureParams
	}

	// encode default
	values := url.Values{}
	for k, v := range config.Template["default"] {
		values.Set(k, fmt.Sprint(v))
	}
	png.SetTemplate("default", png.GetPictureParams(httptest.NewRequest("GET", "/?"+values.Encode(), nil), nil))

	for templateName, templateData := range config.Template {
		if templateName == "default" {
			continue
		}
		values := url.Values{}
		for k, v := range templateData {
			values.Set(k, fmt.Sprint(v))
		}
		png.SetTemplate(templateName, png.GetPictureParams(httptest.NewRequest("GET", "/?"+values.Encode(), nil), nil))
	}

	handler, err := pkg.NewPNG(config.Main.PrometheusAddr, config.Main.PrometheusPath, config.Main.PrometheusAuth, config.Main.CACert, config.Main.Timeout)
	if err != nil {
		log.Fatal(err)
	}
	http.Handle("/", handler)
	log.Fatal(http.ListenAndServe(config.Main.Listen, nil))
}

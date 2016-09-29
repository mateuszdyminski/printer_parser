package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"strings"

	"github.com/BurntSushi/toml"
	"github.com/PuerkitoBio/goquery"
	"github.com/gocarina/gocsv"
	"golang.org/x/net/html"
)

var configPath = flag.String("config", "config.toml", "path to configuration")

func main() {
	// load config
	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Can't load config file. Err: %v \n", err)
		return
	}

	printers := make([]printer, 0, 0)
	for _, page := range cfg.Pages {
		// load content
		resp, err := http.Get(page)
		if err != nil {
			fmt.Printf("Can't fetch http content from page: %s. Err: %v \n", page, err)
			return
		}

		doc, err := goquery.NewDocumentFromResponse(resp)
		if err != nil {
			fmt.Printf("Can't parse html document. Err: %v \n", page, err)
			return
		}

		selection := doc.Find(".productListing tr")

		for _, node := range selection.Nodes {
			printer := printer{}
			innerPrinter := goquery.NewDocumentFromNode(node)

			productTd := innerPrinter.Find("td")
			for i := 0; i < len(productTd.Nodes); i++ {
				// if i == 1 <- get link and the name of the printer, ean, serial number
				if i == 1 {
					linkToPrinter := goquery.NewDocumentFromNode(productTd.Nodes[i]).Find("a")
					printer.Name = linkToPrinter.Nodes[0].FirstChild.Data
					printer.Link = getAttr("href", linkToPrinter.Nodes[0].Attr)

					printerDescription := goquery.NewDocumentFromNode(productTd.Nodes[i]).Find("div")

					number := printerDescription.Nodes[0].FirstChild.Data
					if strings.Contains(number, "Artikelnr.: ") {
						number = strings.TrimLeft(number, "Artikelnr.: ")
						printer.Number = number
					}

					ean := printerDescription.Nodes[0].LastChild.Data
					if strings.Contains(ean, "EAN Code: ") {
						ean = strings.TrimLeft(ean, "EAN Code: ")
						printer.Ean = ean
					}
				}

				// if i == 2 <- get price
				if i == 2 {
					priceImgs := goquery.NewDocumentFromNode(productTd.Nodes[i]).Find("div div img")
					price := ""
					for _, img := range priceImgs.Nodes {
						if isPriceImg(img.Attr) {
							price += getAttr("alt", img.Attr)
						}

						if isSeparatorImg(img.Attr) {
							price += `.`
						}

						if isCurrencyImg(img.Attr) {
							printer.Currency = getAttr("alt", img.Attr)
						}
					}

					printer.Price = price
				}
			}

			printers = append(printers, printer)
		}
	}

	filename := fmt.Sprintf("output/printers%s.csv", time.Now().Format("2006-01-02_15:04:05"))

	f, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Can't create file. Err: %v \n", err)
		return
	}
	defer f.Close()

	if err := gocsv.MarshalFile(&printers, f); err != nil {
		fmt.Printf("Can't masrshal printers to csv. Err: %v \n", err)
		return
	}

	fmt.Printf("Fetched %d printers. Saved in file: %s \n", len(printers), filename)
}

var priceImgs map[string]bool = map[string]bool{
	"images/price/0.gif": true,
	"images/price/1.gif": true,
	"images/price/2.gif": true,
	"images/price/3.gif": true,
	"images/price/4.gif": true,
	"images/price/5.gif": true,
	"images/price/6.gif": true,
	"images/price/7.gif": true,
	"images/price/8.gif": true,
	"images/price/9.gif": true,
}

var currencyImgs map[string]bool = map[string]bool{
	"images/price/euro.gif":   true,
	"images/price/dollar.gif": true,
}

var separatorsImgs map[string]bool = map[string]bool{
	"images/price/komma.gif": true,
}

func isSeparatorImg(attributes []html.Attribute) bool {
	src := getAttr("src", attributes)

	_, ok := separatorsImgs[src]

	return ok
}

func isPriceImg(attributes []html.Attribute) bool {
	src := getAttr("src", attributes)

	_, ok := priceImgs[src]

	return ok
}

func isCurrencyImg(attributes []html.Attribute) bool {
	src := getAttr("src", attributes)

	_, ok := currencyImgs[src]

	return ok
}

func getAttr(attr string, attributes []html.Attribute) string {
	for _, val := range attributes {
		if val.Key == attr {
			return val.Val
		}
	}

	return ""
}

type printer struct {
	Name     string `csv:"name"`
	Link     string `csv:"link"`
	Number   string `csv:"number"`
	Ean      string `csv:"ean"`
	Price    string `csv:"price"`
	Currency string `csv:"currency"`
}

type config struct {
	Pages []string
}

func loadConfig() (*config, error) {
	flag.Parse()
	conf := config{}

	// load config from file
	if *configPath != "" {
		bytes, err := ioutil.ReadFile(*configPath)
		if err != nil {
			return nil, err
		}

		if err := toml.Unmarshal(bytes, &conf); err != nil {
			return nil, err
		}
	}

	return &conf, nil
}

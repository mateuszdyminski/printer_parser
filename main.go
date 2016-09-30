package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

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

	products := make([]product, 0, 0)
	currentPage := 1
	continueWork := true
	for continueWork {
		url := fmt.Sprintf("%s%d", cfg.RootPage, currentPage)
		// load content
		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("Can't fetch http content from page: %s. Err: %v \n", url, err)
			return
		}

		doc, err := goquery.NewDocumentFromResponse(resp)
		if err != nil {
			fmt.Printf("Can't parse html document. Err: %v \n", url, err)
			return
		}

		selection := doc.Find(".productListing tr")

		for _, node := range selection.Nodes {
			product := product{}
			innerProduct := goquery.NewDocumentFromNode(node)

			productTd := innerProduct.Find("td")
			for i := 0; i < len(productTd.Nodes); i++ {
				// if i == 1 <- get link and the name of the product, ean, serial number
				if i == 1 {
					linkToProduct := goquery.NewDocumentFromNode(productTd.Nodes[i]).Find("a")
					product.Name = linkToProduct.Nodes[0].FirstChild.Data
					product.Link = getAttr("href", linkToProduct.Nodes[0].Attr)

					productDescription := goquery.NewDocumentFromNode(productTd.Nodes[i]).Find("div")


					node := productDescription.Nodes[0].FirstChild
					for node != nil {
						value := node.Data
						if strings.Contains(value, "Herstellernr.: ") {
							value = strings.TrimLeft(value, "Herstellernr.: ")
							product.ManufacturerNumber = value
						}

						if strings.Contains(value, "Artikelnr.: ") {
							value = strings.TrimLeft(value, "Artikelnr.: ")
							product.Ean = value
						}

						if strings.Contains(value, "EAN Code: ") {
							value = strings.TrimLeft(value, "EAN Code: ")
							product.Ean = value
						}

						node = node.NextSibling
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
							product.Currency = getAttr("alt", img.Attr)
						}
					}

					product.Price = price
				}
			}

			products = append(products, product)
		}

		if len(selection.Nodes) != 10 {
			continueWork = false
		} else {

			currentPage += 1
		}

		fmt.Printf("Fetched and parsed %d products. Total number: %d\n", len(selection.Nodes), len(products))
	}

	filename := fmt.Sprintf("output/products%s.csv", time.Now().Format("2006-01-02_15:04:05"))

	f, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Can't create file. Err: %v \n", err)
		return
	}
	defer f.Close()

	if err := gocsv.MarshalFile(&products, f); err != nil {
		fmt.Printf("Can't masrshal products to csv. Err: %v \n", err)
		return
	}

	fmt.Printf("Fetched %d products. Saved in file: %s \n", len(products), filename)
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

type product struct {
	Name     string `csv:"name"`
	Link     string `csv:"link"`
	Number   string `csv:"number"`
	Ean      string `csv:"ean"`
	ManufacturerNumber      string `csv:"manufacturerNumber"`
	Price    string `csv:"price"`
	Currency string `csv:"currency"`
}

type config struct {
	RootPage string
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

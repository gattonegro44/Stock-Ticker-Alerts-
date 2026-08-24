// stock_alerts.go
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
	"github.com/google/uuid"
)

type Alert struct {
	ID        string  `json:"id"`
	Symbol    string  `json:"symbol"`
	Threshold float64 `json:"threshold"`
	Above     bool    `json:"above"`
	Triggered bool    `json:"triggered"`
}

type App struct {
	Alerts []Alert `json:"alerts"`
	File   string
}

func NewApp(file string) *App {
	a := &App{File: file}
	a.load()
	return a
}

func (a *App) load() {
	data, err := os.ReadFile(a.File)
	if err != nil {
		return
	}
	json.Unmarshal(data, a)
}

func (a *App) save() {
	data, _ := json.MarshalIndent(a, "", "  ")
	os.WriteFile(a.File, data, 0644)
}

func (a *App) getAlert(symbol string) *Alert {
	for i := range a.Alerts {
		if a.Alerts[i].Symbol == strings.ToUpper(symbol) {
			return &a.Alerts[i]
		}
	}
	return nil
}

func (a *App) add(symbol string, threshold float64, above bool) {
	symbol = strings.ToUpper(symbol)
	// Remove existing
	a.Alerts = []Alert{}
	for _, al := range a.Alerts {
		if al.Symbol != symbol {
			a.Alerts = append(a.Alerts, al)
		}
	}
	alert := Alert{
		ID:        uuid.New().String()[:8],
		Symbol:    symbol,
		Threshold: threshold,
		Above:     above,
		Triggered: false,
	}
	a.Alerts = append(a.Alerts, alert)
	a.save()
	fmt.Printf("✅ Added alert: %s %s %.2f\n", symbol, func() string {
		if above { return ">" } else { return "<" }
	}(), threshold)
}

func (a *App) remove(symbol string) {
	symbol = strings.ToUpper(symbol)
	newAlerts := []Alert{}
	for _, al := range a.Alerts {
		if al.Symbol != symbol {
			newAlerts = append(newAlerts, al)
		}
	}
	a.Alerts = newAlerts
	a.save()
	fmt.Printf("✅ Removed alert for %s\n", symbol)
}

func (a *App) list() {
	if len(a.Alerts) == 0 {
		fmt.Println("No alerts.")
		return
	}
	fmt.Println("\n📋 Current alerts:")
	for _, al := range a.Alerts {
		status := "⏳"
		if al.Triggered {
			status = "🔔"
		}
		sym := al.Symbol
		op := ">"
		if !al.Above {
			op = "<"
		}
		fmt.Printf("  %s: %s %.2f %s\n", sym, op, al.Threshold, status)
	}
}

func fetchPrice(symbol string) (float64, error) {
	url := fmt.Sprintf("https://www.alphavantage.co/query?function=GLOBAL_QUOTE&symbol=%s&apikey=demo", symbol)
	resp, err := http.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return 0, err
	}
	quote, ok := data["Global Quote"].(map[string]interface{})
	if !ok {
		return 0, fmt.Errorf("no quote data")
	}
	priceStr, ok := quote["05. price"].(string)
	if !ok {
		return 0, fmt.Errorf("price not found")
	}
	return strconv.ParseFloat(priceStr, 64)
}

func (a *App) checkAlerts() {
	for i := range a.Alerts {
		price, err := fetchPrice(a.Alerts[i].Symbol)
		if err != nil {
			fmt.Printf("Error fetching %s: %v\n", a.Alerts[i].Symbol, err)
			continue
		}
		if a.Alerts[i].Above && price > a.Alerts[i].Threshold {
			if !a.Alerts[i].Triggered {
				a.Alerts[i].Triggered = true
				a.save()
				fmt.Printf("\n🔔 ALERT: %s price is %.2f (above %.2f)!\n", a.Alerts[i].Symbol, price, a.Alerts[i].Threshold)
			}
		} else if !a.Alerts[i].Above && price < a.Alerts[i].Threshold {
			if !a.Alerts[i].Triggered {
				a.Alerts[i].Triggered = true
				a.save()
				fmt.Printf("\n🔔 ALERT: %s price is %.2f (below %.2f)!\n", a.Alerts[i].Symbol, price, a.Alerts[i].Threshold)
			}
		}
	}
}

func (a *App) watch(interval int) {
	fmt.Printf("📈 Monitoring alerts every %ds (press Ctrl+C to stop)\n", interval)
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			a.checkAlerts()
		}
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: stock_alerts <command> [options]")
		return
	}
	app := NewApp("alerts.json")
	cmd := os.Args[1]

	switch cmd {
	case "add":
		addCmd := flag.NewFlagSet("add", flag.ExitOnError)
		symbol := addCmd.String("symbol", "", "")
		threshold := addCmd.Float64("threshold", 0, "")
		above := addCmd.Bool("above", true, "")
		below := addCmd.Bool("below", false, "")
		addCmd.Parse(os.Args[2:])
		if *symbol == "" && len(addCmd.Args()) > 0 {
			*symbol = addCmd.Args()[0]
		}
		if *threshold == 0 && len(addCmd.Args()) > 1 {
			*threshold, _ = strconv.ParseFloat(addCmd.Args()[1], 64)
		}
		if *threshold == 0 {
			fmt.Println("threshold required")
			return
		}
		aboveFlag := *above
		if *below {
			aboveFlag = false
		}
		app.add(*symbol, *threshold, aboveFlag)

	case "remove":
		if len(os.Args) < 3 {
			fmt.Println("remove <symbol>")
			return
		}
		app.remove(os.Args[2])

	case "list":
		app.list()

	case "watch":
		watchCmd := flag.NewFlagSet("watch", flag.ExitOnError)
		interval := watchCmd.Int("interval", 60, "")
		watchCmd.Parse(os.Args[2:])
		app.watch(*interval)

	default:
		fmt.Println("Unknown command. Use add, remove, list, watch.")
	}
}

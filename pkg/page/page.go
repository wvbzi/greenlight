package page

import (
	"fmt"
	"log"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type BrowserInterface interface {
	SendCommandWithoutResponse(method string, params map[string]interface{}) error
	SendCommandWithResponse(method string, params map[string]interface{}) (map[string]interface{}, error)
}

type Page struct {
	browser BrowserInterface
}

type Locator struct {
	page     *Page
	selector string
}

func NewPage(browser BrowserInterface) *Page {
	return &Page{browser: browser}
}

func (p *Page) Locator(selector string) *Locator {
	return &Locator{
		page:     p,
		selector: selector,
	}
}

func (p *Page) YellowLight(milliseconds int) {
	time.Sleep(time.Duration(milliseconds) * time.Millisecond)
}

func (p *Page) Goto(url string) {
	log.Printf("Navigating to: %s", url)

	if err := p.browser.SendCommandWithoutResponse("Page.enable", nil); err != nil {
		log.Fatalf("Failed to enable Page domain: %v", err)
	}
	if err := p.browser.SendCommandWithoutResponse("Network.enable", nil); err != nil {
		log.Fatalf("Failed to enable Network domain: %v", err)
	}

	params := map[string]interface{}{
		"url": url,
	}
	if err := p.browser.SendCommandWithoutResponse("Page.navigate", params); err != nil {
		log.Fatalf("Failed to navigate to %s: %v", url, err)
	}

	log.Printf("Successfully navigated to: %s", url)
}

func (l *Locator) elementExists() (bool, error) {
	params := map[string]interface{}{
		"expression":    fmt.Sprintf(`document.querySelector("%s") !== null`, l.selector),
		"returnByValue": true,
	}
	response, err := l.page.browser.SendCommandWithResponse("Runtime.evaluate", params)
	if err != nil {
		return false, err
	}

	if result, ok := response["result"].(map[string]interface{}); ok {
		if nestedResult, ok := result["result"].(map[string]interface{}); ok {
			if value, ok := nestedResult["value"].(bool); ok {
				return value, nil
			}
		}
	}
	return false, fmt.Errorf("unexpected response format: %v", response)
}

func (l *Locator) Fill(value string) {
	timeout := 30 * time.Second
	interval := 350 * time.Millisecond
	startTime := time.Now()

	for {
		if time.Since(startTime) > timeout {
			log.Fatalf("Timeout exceeded while waiting for selector: %s", l.selector)
		}

		exists, err := l.elementExists()
		if err != nil {
			time.Sleep(interval)
			continue
		}

		if exists {
			// Focus element
			l.page.browser.SendCommandWithoutResponse("Runtime.evaluate", map[string]interface{}{
				"expression": fmt.Sprintf(`document.querySelector("%s").focus()`, l.selector),
			})

			l.page.browser.SendCommandWithoutResponse("Input.dispatchKeyEvent", map[string]interface{}{
				"type":      "keyDown",
				"modifiers": 2,
				"key":       "a",
			})
			l.page.browser.SendCommandWithoutResponse("Input.dispatchKeyEvent", map[string]interface{}{
				"type": "keyDown",
				"key":  "Backspace",
			})

			l.page.browser.SendCommandWithoutResponse("Input.insertText", map[string]interface{}{
				"text": value,
			})

			log.Printf("Filled selector %s with value: %s", l.selector, value)
			return
		}
		time.Sleep(interval)
	}
}

func (l *Locator) Click() {
	timeout := 30 * time.Second
	interval := 350 * time.Millisecond
	startTime := time.Now()

	for {
		if time.Since(startTime) > timeout {
			log.Fatalf("Timeout exceeded while waiting for selector: %s", l.selector)
		}

		exists, err := l.elementExists()
		if err != nil {
			time.Sleep(interval)
			continue
		}

		if exists {
			params := map[string]interface{}{
				"expression":   fmt.Sprintf(`document.querySelector("%s").click()`, l.selector),
				"awaitPromise": true,
			}
			if err := l.page.browser.SendCommandWithoutResponse("Runtime.evaluate", params); err != nil {
				log.Fatalf("Failed to click on selector %s: %v", l.selector, err)
			}
			log.Printf("Clicked on selector: %s", l.selector)
			return
		}
		time.Sleep(interval)
	}
}

func (l *Locator) TypeSequentially(text string, delayMs int) {
	timeout := 30 * time.Second
	interval := 350 * time.Millisecond
	startTime := time.Now()

	for {
		if time.Since(startTime) > timeout {
			log.Fatalf("Timeout exceeded while waiting for selector: %s", l.selector)
		}

		exists, err := l.elementExists()
		if err != nil {
			time.Sleep(interval)
			continue
		}

		if exists {

			l.page.browser.SendCommandWithoutResponse("Runtime.evaluate", map[string]interface{}{
				"expression": fmt.Sprintf(`document.querySelector("%s").focus()`, l.selector),
			})

			for _, char := range text {
				l.page.browser.SendCommandWithoutResponse("Input.insertText", map[string]interface{}{
					"text": string(char),
				})
				time.Sleep(time.Duration(delayMs) * time.Millisecond)
			}
			return
		}
		time.Sleep(interval)
	}
}

func (l *Locator) InnerText() string {
	timeout := 30 * time.Second
	interval := 350 * time.Millisecond
	startTime := time.Now()

	for {
		if time.Since(startTime) > timeout {
			log.Fatalf("Timeout exceeded while waiting for selector: %s", l.selector)
		}

		exists, err := l.elementExists()
		if err != nil {
			time.Sleep(interval)
			continue
		}

		if exists {
			params := map[string]interface{}{
				"expression":    fmt.Sprintf(`document.querySelector("%s").innerText`, l.selector),
				"returnByValue": true,
			}
			response, err := l.page.browser.SendCommandWithResponse("Runtime.evaluate", params)
			if err != nil {
				log.Fatalf("Failed to get inner text for selector %s: %v", l.selector, err)
			}
			if result, ok := response["result"].(map[string]interface{}); ok {
				if value, ok := result["value"].(string); ok {
					return value
				}
			}
			log.Fatalf("Unexpected response format for inner text: %v", response)
		}
		time.Sleep(interval)
	}
}

func (l *Locator) TypeWithMistakes(text string, delayMs int) {
	timeout := 30 * time.Second
	interval := 350 * time.Millisecond
	startTime := time.Now()

	for {
		if time.Since(startTime) > timeout {
			log.Fatalf("Timeout exceeded while waiting for selector: %s", l.selector)
		}

		exists, err := l.elementExists()
		if err != nil {
			time.Sleep(interval)
			continue
		}

		if exists {
			l.page.browser.SendCommandWithoutResponse("Runtime.evaluate", map[string]interface{}{
				"expression": fmt.Sprintf(`document.querySelector("%s").focus()`, l.selector),
			})

			for _, char := range text {
				if rand.Float32() < 0.4 {
					wrongChar := string(rand.Int31n(26) + 'a')

					l.page.browser.SendCommandWithoutResponse("Input.insertText", map[string]interface{}{
						"text": wrongChar,
					})
					time.Sleep(time.Duration(delayMs) * time.Millisecond)

					l.page.browser.SendCommandWithoutResponse("Input.dispatchKeyEvent", map[string]interface{}{
						"type":                  "rawKeyDown",
						"key":                   "Backspace",
						"windowsVirtualKeyCode": 8,
						"nativeVirtualKeyCode":  8,
					})
					time.Sleep(time.Duration(delayMs) * time.Millisecond)
				}

				l.page.browser.SendCommandWithoutResponse("Input.insertText", map[string]interface{}{
					"text": string(char),
				})
				time.Sleep(time.Duration(delayMs) * time.Millisecond)
			}
			return
		}
		time.Sleep(interval)
	}
}

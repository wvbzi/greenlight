package page

import (
	"context"
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
	ctx     context.Context
}

type Locator struct {
	page     *Page
	selector string
}

func NewPage(ctx context.Context, browser BrowserInterface) *Page {
	return &Page{browser: browser, ctx: ctx}
}

func (p *Page) Locator(selector string) *Locator {
	return &Locator{
		page:     p,
		selector: selector,
	}
}

func (p *Page) YellowLight(milliseconds int) error {
	select {
	case <-p.ctx.Done():
		return p.ctx.Err()

	case <-time.After(time.Duration(milliseconds) * time.Millisecond):
		return nil
	}
}

func (p *Page) Goto(url string) error {
	log.Printf("Navigating to: %s", url)

	if err := p.browser.SendCommandWithoutResponse("Page.enable", nil); err != nil {
		return fmt.Errorf("Failed to enable Page domain: %v", err)
	}
	if err := p.browser.SendCommandWithoutResponse("Network.enable", nil); err != nil {
		return fmt.Errorf("Failed to enable Network domain: %v", err)
	}

	params := map[string]interface{}{
		"url": url,
	}
	if err := p.browser.SendCommandWithoutResponse("Page.navigate", params); err != nil {
		return fmt.Errorf("Failed to navigate to %s: %v", url, err)
	}

	log.Printf("Successfully navigated to: %s", url)
	return nil
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

func (l *Locator) Fill(value string) error {
	timeout := time.NewTimer(30 * time.Second)
	defer timeout.Stop()
	interval := time.NewTicker(350 * time.Millisecond)
	defer interval.Stop()

	for {
		select {
		case <-l.page.ctx.Done():
			return fmt.Errorf("Operation cancelled: %s", l.page.ctx.Err())

		case <-timeout.C:
			return fmt.Errorf("Timeout exceeded while waiting for selector: %s", l.selector)

		case <-interval.C:
			exists, err := l.elementExists()
			if err != nil {
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
				return nil
			}
		}
	}
}

func (l *Locator) Click() error {
	timeout := time.NewTimer(30 * time.Second)
	defer timeout.Stop()
	interval := time.NewTicker(350 * time.Millisecond)
	defer interval.Stop()

	for {
		select {
		case <-l.page.ctx.Done():
			return fmt.Errorf("Operation cancelled: %s", l.page.ctx.Err())

		case <-timeout.C:
			return fmt.Errorf("Timeout exceeded while waiting for selector: %s", l.selector)

		case <-interval.C:
			exists, err := l.elementExists()
			if err != nil {
				continue
			}
			if exists {
				params := map[string]interface{}{
					"expression":   fmt.Sprintf(`document.querySelector("%s").click()`, l.selector),
					"awaitPromise": true,
				}
				if err := l.page.browser.SendCommandWithoutResponse("Runtime.evaluate", params); err != nil {
					return fmt.Errorf("Failed to click on selector %s: %v", l.selector, err)
				}
				log.Printf("Clicked on selector: %s", l.selector)
				return nil
			}
		}
	}
}

func (l *Locator) TypeSequentially(text string, delayMs int) error {
	timeout := time.NewTimer(30 * time.Second)
	defer timeout.Stop()
	interval := time.NewTicker(350 * time.Millisecond)
	defer interval.Stop()

	for {
		select {
		case <-l.page.ctx.Done():
			return fmt.Errorf("Operation cancelled: %s", l.page.ctx.Err())

		case <-timeout.C:
			return fmt.Errorf("Timeout exceeded while waiting for selector: %s", l.selector)

		case <-interval.C:
			exists, err := l.elementExists()
			if err != nil {
				continue
			}
			if exists {
				l.page.browser.SendCommandWithoutResponse("Runtime.evaluate", map[string]interface{}{
					"expression": fmt.Sprintf(`document.querySelector("%s").focus()`, l.selector),
				})

				for _, char := range text {
					select {
					case <-l.page.ctx.Done():
						return fmt.Errorf("")

					case <-time.After(time.Duration(delayMs) * time.Millisecond):
						l.page.browser.SendCommandWithoutResponse("Input.insertText", map[string]interface{}{
							"text": string(char),
						})
					}
				}
				return nil
			}
		}
	}
}

func (l *Locator) InnerText() (string, error) {
	timeout := time.NewTimer(30 * time.Second)
	defer timeout.Stop()
	interval := time.NewTicker(350 * time.Millisecond)
	defer interval.Stop()

	for {
		select {
		case <-l.page.ctx.Done():
			return "", fmt.Errorf("Operation cancelled: %s", l.page.ctx.Err())

		case <-timeout.C:
			return "", fmt.Errorf("Timeout exceeded while waiting for selector: %s", l.selector)

		case <-interval.C:
			exists, err := l.elementExists()
			if err != nil {
				continue
			}
			if exists {
				params := map[string]interface{}{
					"expression":    fmt.Sprintf(`document.querySelector("%s").innerText`, l.selector),
					"returnByValue": true,
				}
				response, err := l.page.browser.SendCommandWithResponse("Runtime.evaluate", params)
				if err != nil {
					return "", fmt.Errorf("Failed to get inner text for selector %s: %v", l.selector, err)
				}
				if result, ok := response["result"].(map[string]interface{}); ok {
					if nestedResult, ok := result["result"].(map[string]interface{}); ok {
						if value, ok := nestedResult["value"].(string); ok {
							return value, nil
						}
					}
				}
				return "", fmt.Errorf("Unexpected response format for inner text: %v", response)
			}
		}
	}
}

func (l *Locator) TypeWithMistakes(text string, delayMs int) error {
	timeout := time.NewTimer(30 * time.Second)
	defer timeout.Stop()
	interval := time.NewTicker(350 * time.Millisecond)
	defer interval.Stop()

	for {
		select {
		case <-l.page.ctx.Done():
			return fmt.Errorf("Operation cancelled: %s", l.page.ctx.Err())

		case <-timeout.C:
			return fmt.Errorf("Timeout exceeded while waiting for selector: %s", l.selector)

		case <-interval.C:
			exists, err := l.elementExists()
			if err != nil {
				continue
			}

			if exists {
				l.page.browser.SendCommandWithoutResponse("Runtime.evaluate", map[string]interface{}{
					"expression": fmt.Sprintf(`document.querySelector("%s").focus()`, l.selector),
				})

				for _, char := range text {
					select {
					case <-l.page.ctx.Done():
						return fmt.Errorf("Operation cancelled: %s", l.page.ctx.Err())

					case <-time.After(time.Duration(delayMs) * time.Millisecond):
						if rand.Float32() < 0.4 {
							wrongChar := string(rand.Int31n(26) + 'a')

							l.page.browser.SendCommandWithoutResponse("Input.insertText", map[string]interface{}{
								"text": wrongChar,
							})

							select {
							case <-l.page.ctx.Done():
								return fmt.Errorf("Operation cancelled: %s", l.page.ctx.Err())

							case <-time.After(time.Duration(delayMs) * time.Millisecond):
								l.page.browser.SendCommandWithoutResponse("Input.dispatchKeyEvent", map[string]interface{}{
									"type":                  "rawKeyDown",
									"key":                   "Backspace",
									"windowsVirtualKeyCode": 8,
									"nativeVirtualKeyCode":  8,
								})
							}

							select {
							case <-l.page.ctx.Done():
								return fmt.Errorf("Operation cancelled: %s", l.page.ctx.Err())

							case <-time.After(time.Duration(delayMs) * time.Millisecond):

							}
						}

						// Sends correct character.
						l.page.browser.SendCommandWithoutResponse("Input.insertText", map[string]interface{}{
							"text": string(char),
						})
					}
				}
				return nil
			}
		}
	}
}

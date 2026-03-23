package browser

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/wvbzi/greenlight/pkg/page"
)

type Browser struct {
	execPath     string
	wsEndpoint   string
	conn         *websocket.Conn
	cmd          *exec.Cmd
	context      context.Context
	cancel       context.CancelFunc
	userDataDir  string
	messageID    int
	messageMutex sync.Mutex
	pid          int
	isHeadless   bool
}

func GreenLight(ctx context.Context, execPath string, isHeadless bool, startURL string) (*Browser, error) {
	ctx, cancel := context.WithCancel(ctx)
	userDataDir := filepath.Join(os.TempDir(), fmt.Sprintf("greenlight_%s", uuid.New().String()))

	browser := &Browser{
		execPath:    execPath,
		context:     ctx,
		cancel:      cancel,
		userDataDir: userDataDir,
		isHeadless:  isHeadless,
	}

	if err := browser.launch(startURL); err != nil {
		return nil, fmt.Errorf("Failed to launch browser: %v", err)
	}

	return browser, nil
}

func (b *Browser) launch(startURL string) error {
	debugPort := "9222"
	args := []string{
		"--remote-debugging-port=" + debugPort,
		"--no-first-run",
		"--user-data-dir=" + b.userDataDir,
		"--remote-allow-origins=*",
		startURL,
	}

	if b.isHeadless {
		args = append(args, "--headless=new")
	}

	b.cmd = exec.CommandContext(b.context, b.execPath, args...)
	if err := b.cmd.Start(); err != nil {
		return fmt.Errorf("Failed to start browser: %v", err)
	}

	b.pid = b.cmd.Process.Pid
	log.Printf("Chrome started with PID: %d", b.pid)

	time.Sleep(time.Second)
	if err := b.attachToPage(); err != nil {
		return err
	}

	return nil
}

func (b *Browser) attachToPage() error {
	debugPort := "9222"
	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/json", debugPort))
	if err != nil {
		return fmt.Errorf("failed to fetch active pages: %v", err)
	}
	defer resp.Body.Close()

	var pages []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&pages); err != nil {
		return fmt.Errorf("failed to decode JSON: %v", err)
	}

	for _, page := range pages {
		if page["type"] == "page" && page["url"] != "" {
			if wsURL, ok := page["webSocketDebuggerUrl"].(string); ok {
				if b.conn != nil {
					b.conn.Close()
				}
				conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
				if err != nil {
					return fmt.Errorf("Failed to connect to page WebSocket: %v", err)
				}
				b.conn = conn
				b.wsEndpoint = wsURL
				log.Printf("Connected to page: %s", page["url"])
				return nil
			}
		}
	}
	return fmt.Errorf("No suitable page found")
}
func (b *Browser) SendCommandWithResponse(method string, params map[string]interface{}) (map[string]interface{}, error) {
	b.messageMutex.Lock()
	b.messageID++
	id := b.messageID
	b.messageMutex.Unlock()

	message := map[string]interface{}{
		"id":     id,
		"method": method,
		"params": params,
	}

	if b.conn == nil {
		if err := b.attachToPage(); err != nil {
			return nil, fmt.Errorf("failed to reconnect WebSocket: %v", err)
		}
	}

	if err := b.conn.WriteJSON(message); err != nil {
		return nil, fmt.Errorf("failed to send WebSocket message: %v", err)
	}

	type wsResult struct {
		response map[string]interface{}
		err      error
	}

	resultChan := make(chan wsResult, 1)

	go func() {
		for {
			_, data, err := b.conn.ReadMessage()
			if err != nil {
				resultChan <- wsResult{nil, fmt.Errorf("failed to read WebSocket message: %v", err)}
				return
			}

			var response map[string]interface{}
			if err := json.Unmarshal(data, &response); err != nil {
				log.Printf("Failed to parse WebSocket message: %s", string(data))
				continue
			}

			if responseID, ok := response["id"].(float64); ok && int(responseID) == id {
				resultChan <- wsResult{response, nil}
				return
			}
		}
	}()

	// Waits until there's a context.cancel() call or until error/valid response from ws.
	select {
	case <-b.context.Done():
		return nil, b.context.Err()

	case result := <-resultChan:
		return result.response, result.err
	}
}

func (b *Browser) SendCommandWithoutResponse(method string, params map[string]interface{}) error {
	b.messageMutex.Lock()
	b.messageID++
	id := b.messageID
	b.messageMutex.Unlock()

	message := map[string]interface{}{
		"id":     id,
		"method": method,
		"params": params,
	}

	if b.conn == nil {
		if err := b.attachToPage(); err != nil {
			return fmt.Errorf("failed to reconnect WebSocket: %v", err)
		}
	}

	if err := b.conn.WriteJSON(message); err != nil {
		return fmt.Errorf("failed to send WebSocket message: %v", err)
	}

	return nil
}

func (b *Browser) NewPage() *page.Page {
	if b.conn == nil {
		log.Fatal("WebSocket connection not established. Cannot create a new page.")
	}
	return page.NewPage(b.context, b)
}

func (b *Browser) RedLight() {
	if b.conn != nil {
		if err := b.conn.Close(); err != nil {
			log.Printf("Error closing WebSocket: %v", err)
		}
	}

	if b.cmd != nil && b.cmd.Process != nil {
		if err := b.cmd.Process.Kill(); err != nil {
			log.Printf("Error killing browser process: %v", err)
		} else {
			b.cmd.Wait()
		}
	}

	if b.userDataDir != "" {
		time.Sleep(500 * time.Millisecond)

		if err := os.RemoveAll(b.userDataDir); err != nil {
			log.Printf("Error removing user data directory: %v", err)
		}
	}

	b.cancel()
	log.Println("Browser closed successfully.")
}

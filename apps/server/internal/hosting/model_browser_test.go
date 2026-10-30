package hosting

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestModelConnectionBrowser(t *testing.T) {
	if os.Getenv("RUN_BROWSER_TESTS") != "1" {
		t.Skip("RUN_BROWSER_TESTS=1 required")
	}
	f := setup(t)
	f.server.Config.ModelEncryptionKey = []byte(strings.Repeat("m", 32))
	app, _ := f.create(t)
	if _, err := f.db.Exec(context.Background(), `UPDATE apps SET visibility='public' WHERE id=$1`, app.ID); err != nil {
		t.Fatal(err)
	}
	f.server.ModelTestHTTPClient = &http.Client{Transport: modelRoundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"choices":[{"message":{"content":"pong"}}]}`))}, nil
	})}
	root, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	bundlePath := filepath.Join(t.TempDir(), "cellapp-sdk.js")
	bundle := exec.CommandContext(ctx, filepath.Join(root, "node_modules/.bin/esbuild"),
		"skills/packages/sdk/src/index.ts", "--bundle", "--platform=browser", "--format=iife", "--global-name=CellappSDK", "--minify", "--outfile="+bundlePath)
	bundle.Dir = root
	if output, err := bundle.CombinedOutput(); err != nil {
		t.Fatalf("bundle SDK for static Agent: %v\n%s", err, output)
	}
	script, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatal(err)
	}
	content := `<!doctype html><html><head><meta name="cellapp-api-version" content="1.2"></head><body><h1>Model Agent</h1><button id="run">Run Agent</button><p id="answer"></p><script>` + strings.ReplaceAll(string(script), "</script>", "<\\/script>") + `</script><script>const cellapp=CellappSDK.createCellappClient();document.getElementById('run').onclick=async()=>{try{const result=await cellapp.chat({messages:[{role:'user',content:'ping'}]});document.getElementById('answer').textContent=result.choices[0].message.content}catch(error){document.getElementById('answer').textContent=error.message}}</script></body></html>`
	deployment, _ := f.stage(t, app, content)
	f.uploadAndPublish(t, app, deployment, content)
	f.server.Web, err = LoadWebFS(filepath.Join(root, "apps/web/dist"))
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewUnstartedServer(nil)
	_, port, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	f.server.Config.ControlOrigin = "https://cellapp.localhost:" + port
	f.server.Config.AppPort = port
	server.Config.Handler = f.server.Handler()
	server.StartTLS()
	defer server.Close()
	session := token()
	if _, err := f.db.Exec(context.Background(), `INSERT INTO browser_sessions(hash,owner_id,expires_at) VALUES($1,$2,now()+interval '1 hour')`, keyed(f.server.Config.Secret, session), f.owner); err != nil {
		t.Fatal(err)
	}
	var hostLabel string
	if err := f.db.QueryRow(context.Background(), `SELECT COALESCE(host_label,id) FROM apps WHERE id=$1`, app.ID).Scan(&hostLabel); err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(ctx, "npm", "exec", "--", "tsx", "apps/web/tests/model-real.ts")
	command.Dir = root
	command.Env = append(os.Environ(), "TEST_MODEL_BROWSER_ORIGIN="+f.server.Config.ControlOrigin, "TEST_MODEL_BROWSER_SESSION="+session, "TEST_MODEL_BROWSER_APP="+app.ID, "TEST_MODEL_BROWSER_APP_URL="+f.server.Config.AppHostURL(hostLabel))
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("real model browser acceptance failed: %v\n%s", err, output)
	}
}

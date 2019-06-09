package remote

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/umputun/remark/backend/app/store"
	"github.com/umputun/remark/backend/app/store/engine"
)

func TestClient_Create(t *testing.T) {
	ts := testServer(t, `{"method":"create","params":[{"id":"123","pid":"","text":"msg","user":{"name":"","id":"","picture":"","admin":false},"locator":{"site":"site","url":"http://example.com/url"},"score":0,"vote":0,"time":"0001-01-01T00:00:00Z"}]}`, `{"result":"12345"}`)
	defer ts.Close()
	c := Client{API: ts.URL, Client: http.Client{}}

	res, err := c.Create(store.Comment{ID: "123", Locator: store.Locator{URL: "http://example.com/url", SiteID: "site"},
		Text: "msg"})
	assert.NoError(t, err)
	assert.Equal(t, "12345", res)
	t.Logf("%v %T", res, res)
}

func TestClient_Get(t *testing.T) {
	ts := testServer(t, `{"method":"get","params":[{"url":"http://example.com/url"},"site"]}`,
		`{"result":{"id":"123","pid":"","text":"msg","delete":true}}`)
	defer ts.Close()
	c := Client{API: ts.URL, Client: http.Client{}}

	res, err := c.Get(store.Locator{URL: "http://example.com/url"}, "site")
	assert.NoError(t, err)
	assert.Equal(t, store.Comment{ID: "123", Text: "msg", Deleted: true}, res)
	t.Logf("%v %T", res, res)
}

func TestClient_GetWithErrorResult(t *testing.T) {
	ts := testServer(t, `{"method":"get","params":[{"url":"http://example.com/url"},"site"]}`, `{"error":"failed"}`)
	defer ts.Close()
	c := Client{API: ts.URL, Client: http.Client{}}

	_, err := c.Get(store.Locator{URL: "http://example.com/url"}, "site")
	assert.EqualError(t, err, "failed")
}

func TestClient_GetWithErrorDecode(t *testing.T) {
	ts := testServer(t, `{"method":"get","params":[{"url":"http://example.com/url"},"site"]}`, ``)
	defer ts.Close()
	c := Client{API: ts.URL, Client: http.Client{}}

	_, err := c.Get(store.Locator{URL: "http://example.com/url"}, "site")
	assert.EqualError(t, err, "failed to decode response for get: EOF")
}

func TestClient_GetWithErrorRemote(t *testing.T) {
	c := Client{API: "http://127.0.0.2", Client: http.Client{Timeout: 10 * time.Millisecond}}

	_, err := c.Get(store.Locator{URL: "http://example.com/url"}, "site")
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), "remote call failed for get:"))
}

func TestClient_FailedStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		t.Logf("req: %s", string(body))
		w.WriteHeader(400)
	}))
	defer ts.Close()
	c := Client{API: ts.URL, Client: http.Client{}}

	_, err := c.Get(store.Locator{URL: "http://example.com/url"}, "site")
	assert.EqualError(t, err, "bad status 400 for get")
}

func TestClient_Update(t *testing.T) {
	ts := testServer(t, `{"method":"update","params":[{"url":"http://example.com/url"},{"id":"123","pid":"","text":"msg","user":{"name":"","id":"","picture":"","admin":false},"locator":{"site":"site123","url":"http://example.com/url"},"score":0,"vote":0,"time":"0001-01-01T00:00:00Z"}]}`, `{}`)
	defer ts.Close()
	c := Client{API: ts.URL, Client: http.Client{}}

	err := c.Update(store.Locator{URL: "http://example.com/url"}, store.Comment{ID: "123",
		Locator: store.Locator{URL: "http://example.com/url", SiteID: "site123"}, Text: "msg"})
	assert.NoError(t, err)

}

func TestClient_Find(t *testing.T) {
	ts := testServer(t, `{"method":"find","params":[{"locator":{"url":"http://example.com/url"},"sort":"-time","since":"0001-01-01T00:00:00Z","limit":10}]}`, `{"result":[{"text":"1"},{"text":"2"}]}`)
	defer ts.Close()
	c := Client{API: ts.URL, Client: http.Client{}}

	res, err := c.Find(engine.FindRequest{Locator: store.Locator{URL: "http://example.com/url"}, Sort: "-time", Limit: 10})
	assert.NoError(t, err)
	assert.Equal(t, []store.Comment{{Text: "1"}, {Text: "2"}}, res)
}

func TestClient_Info(t *testing.T) {
	ts := testServer(t, `{"method":"info","params":[{"locator":{"url":"http://example.com/url"},"limit":10,"skip":5,"ro_age":10}]}`, `{"result":[{"url":"u1","count":22},{"url":"u2","count":33}]}`)
	defer ts.Close()
	c := Client{API: ts.URL, Client: http.Client{}}

	res, err := c.Info(engine.InfoRequest{Locator: store.Locator{URL: "http://example.com/url"},
		Limit: 10, Skip: 5, ReadOnlyAge: 10})
	assert.NoError(t, err)
	assert.Equal(t, []store.PostInfo{{URL: "u1", Count: 22}, {URL: "u2", Count: 33}}, res)
}

func TestClient_Flag(t *testing.T) {
	ts := testServer(t, `{"method":"flag","params":[{"flag":"verified","locator":{"url":"http://example.com/url"}}]}`,
		`{"result":false}`)
	defer ts.Close()
	c := Client{API: ts.URL, Client: http.Client{}}

	res, err := c.Flag(engine.FlagRequest{Locator: store.Locator{URL: "http://example.com/url"}, Flag: engine.Verified})
	assert.NoError(t, err)
	assert.Equal(t, false, res)
}

func TestClient_ListFlag(t *testing.T) {
	ts := testServer(t, `{"method":"list_flags","params":["site_id","blocked"]}`, `{"result":[{"ID":"id1"},{"ID":"id2"}]}`)
	defer ts.Close()
	c := Client{API: ts.URL, Client: http.Client{}}
	res, err := c.ListFlags("site_id", engine.Blocked)
	assert.NoError(t, err)
	assert.Equal(t, []interface{}{map[string]interface{}{"ID": "id1"}, map[string]interface{}{"ID": "id2"}}, res)
}

func TestClient_Count(t *testing.T) {
	ts := testServer(t, `{"method":"count","params":[{"locator":{"url":"http://example.com/url"},"since":"0001-01-01T00:00:00Z"}]}`,
		`{"result":11}`)
	defer ts.Close()
	c := Client{API: ts.URL, Client: http.Client{}}

	res, err := c.Count(engine.FindRequest{Locator: store.Locator{URL: "http://example.com/url"}})
	assert.NoError(t, err)
	assert.Equal(t, 11, res)
}

func TestClient_Delete(t *testing.T) {
	ts := testServer(t, `{"method":"delete","params":[{"locator":{"url":"http://example.com/url"},"del_mode":0}]}`, `{}`)
	defer ts.Close()
	c := Client{API: ts.URL, Client: http.Client{}}

	err := c.Delete(engine.DeleteRequest{Locator: store.Locator{URL: "http://example.com/url"}})
	assert.NoError(t, err)
}

func TestClient_Close(t *testing.T) {
	ts := testServer(t, `{"method":"close","params":null}`, `{}`)
	defer ts.Close()
	c := Client{API: ts.URL, Client: http.Client{}}
	err := c.Close()
	assert.NoError(t, err)
}

func testServer(t *testing.T, req, resp string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, req, string(body))
		t.Logf("req: %s", string(body))
		fmt.Fprintf(w, resp)
	}))
}
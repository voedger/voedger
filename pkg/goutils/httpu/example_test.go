/*
 * Copyright (c) 2025-present unTill Software Development Group B.V.
 * @author Denis Gribanov
 */

package httpu

import (
    "context"
    "fmt"
    "net/http"
    "net/http/httptest"
)

func Example() {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Hello, %s!", r.URL.Query().Get("name"))
    }))
    defer server.Close()

    httpClient, cleanup := NewIHTTPClient()
    defer cleanup()

    resp, err := httpClient.Req(
        context.Background(),
        server.URL+"?name=World",
        "",
        WithMethod("GET"),
        WithHeaders("Content-Type", "application/json"),
    )
    if err != nil {
        panic(err)
    }

    fmt.Println("Response:", resp.Body)
    fmt.Println("Status:", resp.HTTPResp.StatusCode)

    // Output:
    // Response: Hello, World!
    // Status: 200
}
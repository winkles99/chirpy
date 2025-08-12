package main
import (
    "net/http"
)

func main() {
    mux := http.NewServeMux()
    //Readiness endpoint at .healthz
    mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/plain; charset=utf-8")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    })

    fileServer := http.FileServer(http.Dir("."))
    mux.Handle("/app/", http.StripPrefix("/app/", fileServer))

    mux.Handle("/", fileServer)
    server := http.Server{
        Addr:    ":8080",
        Handler: mux,
    }
server.ListenAndServe()
}
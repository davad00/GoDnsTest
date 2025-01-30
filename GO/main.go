package main

import (
    "context"
    "fmt"
    "net"
    "time"
    "sort"
)

type DNSProvider struct {
    Name string
    IP   string
}

type Result struct {
    Provider DNSProvider
    Latency  time.Duration
}

func main() {
    providers := []DNSProvider{
        {"Cloudflare", "1.1.1.1"},
        {"Google", "8.8.8.8"},
        {"Quad9", "9.9.9.9"},
        {"OpenDNS", "208.67.222.222"},
        {"Comodo", "8.26.56.26"},
    }

    results := make([]Result, len(providers))

    for i, provider := range providers {
        latency := measureDNSLatency(provider.IP)
        results[i] = Result{Provider: provider, Latency: latency}
    }

    sort.Slice(results, func(i, j int) bool {
        return results[i].Latency < results[j].Latency
    })

    fmt.Println("DNS Provider Latency Results:")
    for _, result := range results {
        fmt.Printf("%s (%s): %v\n", result.Provider.Name, result.Provider.IP, result.Latency)
    }
}

func measureDNSLatency(dnsServer string) time.Duration {
    resolver := &net.Resolver{
        PreferGo: true,
        Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
            d := net.Dialer{}
            return d.DialContext(ctx, "udp", dnsServer+":53")
        },
    }

    start := time.Now()
    _, err := resolver.LookupHost(context.Background(), "www.example.com")
    if err != nil {
        return time.Hour // Return a large value to indicate failure
    }
    return time.Since(start)
}

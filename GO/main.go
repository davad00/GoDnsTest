package main

import (
    "context"
    "fmt"
    "net"
    "sync"
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

var testDomains = []string{
    "www.google.com",
    "www.amazon.com",
    "www.microsoft.com",
    "www.facebook.com",
    "www.netflix.com",
}

const (
    testsPerDomain = 3  // Number of tests per domain
    timeout        = 5 * time.Second
)

func main() {
    providers := []DNSProvider{
        {"Cloudflare", "1.1.1.1"},
        {"Cloudflare Secondary", "1.0.0.1"},
        {"Google", "8.8.8.8"},
        {"Google Secondary", "8.8.4.4"},
        {"Quad9", "9.9.9.9"},
        {"Quad9 Secondary", "149.112.112.112"},
        {"OpenDNS", "208.67.222.222"},
        {"OpenDNS Secondary", "208.67.220.220"},
        {"Comodo", "8.26.56.26"},
        {"Comodo Secondary", "8.20.247.20"},
        {"AdGuard", "94.140.14.14"},
        {"CleanBrowsing", "185.228.168.9"},
        {"Alternate DNS", "76.76.19.19"},
    }

    results := make([]Result, len(providers))
    var wg sync.WaitGroup
    resultChan := make(chan Result, len(providers))

    // Start concurrent tests for each provider
    for _, provider := range providers {
        wg.Add(1)
        go func(p DNSProvider) {
            defer wg.Done()
            avgLatency := testProvider(p)
            resultChan <- Result{Provider: p, Latency: avgLatency}
        }(provider)
    }

    // Wait for all tests to complete
    go func() {
        wg.Wait()
        close(resultChan)
    }()

    // Collect results
    i := 0
    for result := range resultChan {
        results[i] = result
        i++
    }

    // Sort and display results
    sort.Slice(results, func(i, j int) bool {
        return results[i].Latency < results[j].Latency
    })

    fmt.Println("\nDNS Provider Latency Results (averaged across multiple domains):")
    fmt.Println("--------------------------------------------------------")
    for _, result := range results {
        if result.Latency >= timeout {
            fmt.Printf("%-20s (%s): Timeout or Error\n", result.Provider.Name, result.Provider.IP)
        } else {
            fmt.Printf("%-20s (%s): %v\n", result.Provider.Name, result.Provider.IP, result.Latency)
        }
    }
}

func testProvider(provider DNSProvider) time.Duration {
    var totalLatency time.Duration
    var successfulTests int

    resolver := &net.Resolver{
        PreferGo: true,
        Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
            d := net.Dialer{Timeout: timeout}
            return d.DialContext(ctx, "udp", provider.IP+":53")
        },
    }

    for _, domain := range testDomains {
        for i := 0; i < testsPerDomain; i++ {
            ctx, cancel := context.WithTimeout(context.Background(), timeout)
            start := time.Now()
            _, err := resolver.LookupHost(ctx, domain)
            latency := time.Since(start)
            cancel()

            if err == nil && latency < timeout {
                totalLatency += latency
                successfulTests++
            }
        }
    }

    if successfulTests == 0 {
        return timeout
    }
    return totalLatency / time.Duration(successfulTests)
}

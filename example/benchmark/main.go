package main

import (
	"crypto/md5"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/zeebo/blake3"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run hash_file.go <filename>")
		fmt.Println("Example: go run hash_file.go /path/to/largefile.bin")
		os.Exit(1)
	}

	filename := os.Args[1]

	// Check if file exists
	info, err := os.Stat(filename)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fileSize := info.Size()
	fmt.Println("=" + repeat("=", 69))
	fmt.Println("Hash Benchmark for File")
	fmt.Println("=" + repeat("=", 69))
	fmt.Printf("File: %s\n", filename)
	fmt.Printf("Size: %s (%d bytes)\n", formatSize(fileSize), fileSize)
	fmt.Println()

	fmt.Printf("%-12s %15s %15s %s\n", "Algorithm", "Time", "Throughput", "Hash")
	fmt.Println("-" + repeat("-", 69))

	// Benchmark MD5
	md5Hash, md5Duration := benchmarkFileMD5(filename)
	printFileResult("MD5", md5Duration, fileSize, md5Hash)

	// Benchmark SHA-1
	sha1Hash, sha1Duration := benchmarkFileSHA1(filename)
	printFileResult("SHA-1", sha1Duration, fileSize, sha1Hash)

	// Benchmark BLAKE3
	blake3Hash, blake3Duration := benchmarkFileBLAKE3(filename)
	printFileResult("BLAKE3", blake3Duration, fileSize, blake3Hash)

	fmt.Println()

	// Print comparison
	fmt.Println("Speed comparison:")
	fastest := min(md5Duration, sha1Duration, blake3Duration)
	fmt.Printf("  MD5:    %.2fx\n", float64(md5Duration)/float64(fastest))
	fmt.Printf("  SHA-1:  %.2fx\n", float64(sha1Duration)/float64(fastest))
	fmt.Printf("  BLAKE3: %.2fx\n", float64(blake3Duration)/float64(fastest))
}

func benchmarkFileMD5(filename string) (string, time.Duration) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	hash := md5.New()
	start := time.Now()
	if _, err := io.Copy(hash, file); err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}
	elapsed := time.Since(start)

	return fmt.Sprintf("%x", hash.Sum(nil)), elapsed
}

func benchmarkFileSHA1(filename string) (string, time.Duration) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	hash := sha1.New()
	start := time.Now()
	if _, err := io.Copy(hash, file); err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}
	elapsed := time.Since(start)

	return fmt.Sprintf("%x", hash.Sum(nil)), elapsed
}

func benchmarkFileBLAKE3(filename string) (string, time.Duration) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	hash := blake3.New()
	start := time.Now()
	if _, err := io.Copy(hash, file); err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		os.Exit(1)
	}
	elapsed := time.Since(start)

	return fmt.Sprintf("%x", hash.Sum(nil)), elapsed
}

func printFileResult(name string, duration time.Duration, fileSize int64, hash string) {
	throughput := float64(fileSize) / (1024 * 1024) / duration.Seconds()
	fmt.Printf("%-12s %15s %12.2f MB/s %s\n",
		name,
		duration.Round(time.Microsecond),
		throughput,
		hash,
	)
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func repeat(s string, n int) string {
	result := ""
	for i := 0; i < n; i++ {
		result += s
	}
	return result
}

func min(a, b, c time.Duration) time.Duration {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}

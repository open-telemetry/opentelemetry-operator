// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"time"
)

// getRandomNumber generates a random integer between min and max (inclusive).
func getRandomNumber(min, max int) (int, error) {
	if min > max {
		return 0, fmt.Errorf("min (%d) cannot be greater than max (%d)", min, max)
	}

	// Calculate the range size.
	rangeSize := big.NewInt(int64(max - min + 1))

	// Generate a random number n, where 0 <= n < rangeSize.
	n, err := rand.Int(rand.Reader, rangeSize)
	if err != nil {
		// Return an error if random number generation fails
		return 0, fmt.Errorf("failed to generate random number: %w", err)
	}

	// Convert the big.Int result back to a regular int.
	// Add min to shift the range from [0, rangeSize) to [min, max].
	// n.Int64() is safe here because rangeSize fits in int64 if max-min+1 does.
	result := int(n.Int64()) + min
	return result, nil
}

// rollDiceHandler handles requests to the /rolldice endpoint.
func rollDiceHandler(w http.ResponseWriter, r *http.Request) {
	// Ensure we only handle GET requests for this endpoint
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Generate a random number between 1 and 6
	diceRoll, err := getRandomNumber(1, 6)
	if err != nil {
		// Log the error and return an internal server error if generation failed
		log.Printf("Error generating random number: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Convert the integer result to a string
	responseString := strconv.Itoa(diceRoll)

	// Set the content type header to text/plain
	w.Header().Set("Content-Type", "text/plain")

	// Write the string response back to the client
	_, writeErr := fmt.Fprint(w, responseString)
	if writeErr != nil {
		// Log the error if writing the response failed (response may be partially sent)
		log.Printf("Error writing response: %v", writeErr)
		// Attempt to send an error, though headers might already be sent
		// http.Error(w, "Internal Server Error", http.StatusInternalServerError) // Avoid sending second error
	}
	log.Printf("Rolled a %d for request from %s", diceRoll, r.RemoteAddr) // Optional: Log the roll
}

func main() {
	// Get the port from the environment variable, default to "8095" if not set
	port := os.Getenv("PORT")
	if port == "" {
		port = "8095"
	}

	// Register the handler function for the "/rolldice" path with the DefaultServeMux.
	http.HandleFunc("/rolldice", rollDiceHandler)

	// Construct the server address string (e.g., ":8095")
	serverAddr := ":" + port

	// Configure the HTTP server with timeouts
	server := &http.Server{
		Addr: serverAddr,
		// Use DefaultServeMux by leaving Handler nil, which includes our /rolldice handler
		Handler:      nil,
		ReadTimeout:  10 * time.Second, // Max time to read entire request, including body
		WriteTimeout: 10 * time.Second, // Max time to write response
		IdleTimeout:  60 * time.Second, // Max time for connections using TCP Keep-Alive
	}

	// Print a message indicating the server is starting
	log.Printf("Listening for requests on http://localhost:%s", port)

	// Start the configured HTTP server.
	// This ListenAndServe is now called on our configured server instance.
	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		// Log any fatal errors encountered during server startup, ignore ErrServerClosed
		log.Fatalf("Server failed to start or unexpectedly closed: %v", err)
	} else if err == http.ErrServerClosed {
		log.Println("Server shut down gracefully")
	}
}

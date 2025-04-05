package common

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"testing"
)

func TestCreateEditRequest(t *testing.T) {
	// Create buffer for multipart writer
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add image file
	imageFile, err := os.Open("/Users/dz0400962/Temp/1.jpg")
	if err != nil {
		t.Fatalf("Failed to open image file: %v", err)
	}
	defer imageFile.Close()

	imagePart, err := writer.CreateFormFile("image_file", "1.jpg")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	_, err = io.Copy(imagePart, imageFile)
	if err != nil {
		t.Fatalf("Failed to copy image: %v", err)
	}

	// Add mask file
	maskFile, err := os.Open("/Users/dz0400962/Temp/2.png")
	if err != nil {
		t.Fatalf("Failed to open mask file: %v", err)
	}
	defer maskFile.Close()

	maskPart, err := writer.CreateFormFile("mask", "2.png")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	_, err = io.Copy(maskPart, maskFile)
	if err != nil {
		t.Fatalf("Failed to copy mask: %v", err)
	}

	// Add other form fields
	err = writer.WriteField("prompt", "prompt")
	if err != nil {
		t.Fatalf("Failed to write prompt field: %v", err)
	}

	err = writer.WriteField("model", "V_1")
	if err != nil {
		t.Fatalf("Failed to write model field: %v", err)
	}

	writer.Close()

	// Create request
	req, err := http.NewRequest("POST", "https://api.ideogram.ai/edit", body)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Set headers
	req.Header.Set("Api-Key", "Xctio1j83aD8eMYl3U0eaksiMh8N60kZKxr7MetXjrX-WG4tbhbOZaPmyWozwdQLCG6gtKBjWJBUG13k09aAeQ")
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Print request details for verification
	t.Logf("Request Method: %s", req.Method)
	t.Logf("Request URL: %s", req.URL.String())
	t.Logf("Request Headers: %+v", req.Header)

	// Optional: Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}
	defer resp.Body.Close()
	t.Logf("Response Status: %s", resp.Status)
}

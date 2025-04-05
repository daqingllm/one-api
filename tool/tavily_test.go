package tool

import "testing"

func TestGetAdaptor(t *testing.T) {
	tavily, err := SearchByTavily("who is Leo Messi?")
	if err != nil {
		t.Errorf("SearchByTavily failed: %v", err)
		return
	}
	if tavily == nil {
		t.Errorf("SearchByTavily failed: nil response")
		return
	}
	println(tavily.Answer)
	for _, result := range tavily.Results {
		println(result.Title)
		println(result.Url)
		println(result.Content)
	}
}

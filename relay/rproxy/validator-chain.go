package rproxy

import (
	"sync"

	"github.com/songquanpeng/one-api/relay/model"
)

type ValidatorChain struct {
	Validators []Validator
}

func NewValidatorChain() *ValidatorChain {
	return &ValidatorChain{
		Validators: make([]Validator, 0),
	}
}

func (vc *ValidatorChain) AddValidator(v Validator) *ValidatorChain {
	vc.Validators = append(vc.Validators, v)
	return vc
}

func (vc *ValidatorChain) AddValidators(validators ...Validator) *ValidatorChain {
	vc.Validators = append(vc.Validators, validators...)
	return vc
}

func (vc *ValidatorChain) Validate() *model.ErrorWithStatusCode {
	var wg sync.WaitGroup
	errChan := make(chan *model.ErrorWithStatusCode, len(vc.Validators))
	for _, validator := range vc.Validators {
		wg.Add(1)
		go func(v Validator) {
			defer wg.Done()
			if err := v.Validate(); err != nil {
				errChan <- err
			}
		}(validator)
	}
	wg.Wait()
	close(errChan)
	for err := range errChan {
		return err
	}
	return nil
}

func (vc *ValidatorChain) GetValidators() []Validator {
	return vc.Validators
}

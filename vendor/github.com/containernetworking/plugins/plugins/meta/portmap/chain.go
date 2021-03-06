// Copyright 2017 CNI authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"strings"

	"github.com/coreos/go-iptables/iptables"
	shellwords "github.com/mattn/go-shellwords"
)

type chain struct {
	table       string
	name        string
	entryRule   []string // the rule that enters this chain
	entryChains []string // the chains to add the entry rule
}

// setup idempotently creates the chain. It will not error if the chain exists.
func (c *chain) setup(ipt *iptables.IPTables, rules [][]string) error {
	// create the chain
	exists, err := chainExists(ipt, c.table, c.name)
	if err != nil {
		return err
	}
	if !exists {
		if err := ipt.NewChain(c.table, c.name); err != nil {
			return err
		}
	}

	// Add the rules to the chain
	for i := len(rules) - 1; i >= 0; i-- {
		if err := prependUnique(ipt, c.table, c.name, rules[i]); err != nil {
			return err
		}
	}

	// Add the entry rules
	entryRule := append(c.entryRule, "-j", c.name)
	for _, entryChain := range c.entryChains {
		if err := prependUnique(ipt, c.table, entryChain, entryRule); err != nil {
			return err
		}
	}

	return nil
}

// teardown idempotently deletes a chain. It will not error if the chain doesn't exist.
// It will first delete all references to this chain in the entryChains.
func (c *chain) teardown(ipt *iptables.IPTables) error {
	// flush the chain
	// This will succeed *and create the chain* if it does not exist.
	// If the chain doesn't exist, the next checks will fail.
	if err := ipt.ClearChain(c.table, c.name); err != nil {
		return err
	}

	for _, entryChain := range c.entryChains {
		entryChainRules, err := ipt.List(c.table, entryChain)
		if err != nil {
			// Swallow error here - probably the chain doesn't exist.
			// If we miss something the deletion will fail
			continue
		}

		for _, entryChainRule := range entryChainRules[1:] {
			if strings.HasSuffix(entryChainRule, "-j "+c.name) {
				chainParts, err := shellwords.Parse(entryChainRule)
				if err != nil {
					return fmt.Errorf("error parsing iptables rule: %s: %v", entryChainRule, err)
				}
				chainParts = chainParts[2:] // List results always include an -A CHAINNAME

				if err := ipt.Delete(c.table, entryChain, chainParts...); err != nil {
					return fmt.Errorf("Failed to delete referring rule %s %s: %v", c.table, entryChainRule, err)
				}
			}
		}
	}

	if err := ipt.DeleteChain(c.table, c.name); err != nil {
		return err
	}
	return nil
}

// prependUnique will prepend a rule to a chain, if it does not already exist
func prependUnique(ipt *iptables.IPTables, table, chain string, rule []string) error {
	exists, err := ipt.Exists(table, chain, rule...)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	return ipt.Insert(table, chain, 1, rule...)
}

func chainExists(ipt *iptables.IPTables, tableName, chainName string) (bool, error) {
	chains, err := ipt.ListChains(tableName)
	if err != nil {
		return false, err
	}

	for _, ch := range chains {
		if ch == chainName {
			return true, nil
		}
	}
	return false, nil
}

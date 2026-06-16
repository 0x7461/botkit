package bot

import "fmt"

// NamedSource pairs a Source with the label stamped onto its items.
type NamedSource struct {
	Name   string
	Source Source
}

// MultiSource fans out to several sources and concatenates their items, tagging
// each with Meta["source"] = the sub-source's Name. A sub-source that errors is
// logged and skipped so one failure doesn't sink the whole digest.
type MultiSource struct {
	Sources []NamedSource
}

func (m *MultiSource) Fetch() ([]Item, error) {
	var all []Item
	for _, ns := range m.Sources {
		items, err := ns.Source.Fetch()
		if err != nil {
			fmt.Printf("multisource: skipping %s: %v\n", ns.Name, err)
			continue
		}
		for i := range items {
			if items[i].Meta == nil {
				items[i].Meta = map[string]string{}
			}
			items[i].Meta["source"] = ns.Name
		}
		all = append(all, items...)
	}
	return all, nil
}

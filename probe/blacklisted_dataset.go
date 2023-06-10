// Code generated by go run github.com/nfx/slrp/ql/generator/main.go Foo. DO NOT EDIT.
package probe

import (
	"github.com/nfx/slrp/ql/eval"
)

type blacklistedDataset []blacklisted

func (d blacklistedDataset) Query(query string) (*eval.QueryResult[blacklisted], error) {
	return (&eval.Dataset[blacklisted, blacklistedDataset]{
		Source: d,
		Accessors: eval.Accessors{
			"Proxy":    eval.StringGetter{Name: "Proxy", Func: d.getProxy},
			"Country":  eval.StringGetter{Name: "Country", Func: d.getCountry},
			"Provider": eval.StringGetter{Name: "Provider", Func: d.getProvider},
			"ASN":      eval.NumberGetter{Name: "ASN", Func: d.getASN},
			"Failure":  eval.StringGetter{Name: "Failure", Func: d.getFailure},
		},
		Sorters: eval.Sorters[blacklisted]{
			"Proxy":    {Asc: d.sortAscProxy, Desc: d.sortDescProxy},
			"Country":  {Asc: d.sortAscCountry, Desc: d.sortDescCountry},
			"Provider": {Asc: d.sortAscProvider, Desc: d.sortDescProvider},
			"ASN":      {Asc: d.sortAscASN, Desc: d.sortDescASN},
			"Failure":  {Asc: d.sortAscFailure, Desc: d.sortDescFailure},
		},
		Facets: func(filtered blacklistedDataset, topN int) []eval.Facet {
			return eval.FacetRetrievers[blacklisted]{
				eval.StringFacet{
					Getter: filtered.getCountry,
					Field:  "Country",
					Name:   "Country",
				}, eval.StringFacet{
					Getter: filtered.getProvider,
					Field:  "Provider",
					Name:   "Provider",
				}, eval.StringFacet{
					// TODO: add as an override feature to generator
					Getter: filtered.getFailureFacet,
					// TODO: add as an override feature to generator
					Contains: true,

					Field: "Failure",
					Name:  "Failure",
				},
			}.Facets(filtered, topN)
		},
	}).Query(query)
}

func (d blacklistedDataset) getProxy(record int) string {
	return d[record].Proxy.String()
}

func (_ blacklistedDataset) sortAscProxy(left, right blacklisted) bool {
	return left.Proxy.String() < right.Proxy.String()
}

func (_ blacklistedDataset) sortDescProxy(left, right blacklisted) bool {
	return left.Proxy.String() > right.Proxy.String()
}

func (d blacklistedDataset) getCountry(record int) string {
	return d[record].Country
}

func (_ blacklistedDataset) sortAscCountry(left, right blacklisted) bool {
	return left.Country < right.Country
}

func (_ blacklistedDataset) sortDescCountry(left, right blacklisted) bool {
	return left.Country > right.Country
}

func (d blacklistedDataset) getProvider(record int) string {
	return d[record].Provider
}

func (_ blacklistedDataset) sortAscProvider(left, right blacklisted) bool {
	return left.Provider < right.Provider
}

func (_ blacklistedDataset) sortDescProvider(left, right blacklisted) bool {
	return left.Provider > right.Provider
}

func (d blacklistedDataset) getASN(record int) float64 {
	return float64(d[record].ASN)
}

func (_ blacklistedDataset) sortAscASN(left, right blacklisted) bool {
	return left.ASN < right.ASN
}

func (_ blacklistedDataset) sortDescASN(left, right blacklisted) bool {
	return left.ASN > right.ASN
}

func (d blacklistedDataset) getFailure(record int) string {
	return d[record].Failure
}

func (_ blacklistedDataset) sortAscFailure(left, right blacklisted) bool {
	return left.Failure < right.Failure
}

func (_ blacklistedDataset) sortDescFailure(left, right blacklisted) bool {
	return left.Failure > right.Failure
}

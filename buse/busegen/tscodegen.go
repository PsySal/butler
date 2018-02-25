package main

import (
	"os/exec"
	"strings"
	"time"
)

func (bc *BuseContext) GenerateTsCode() error {
	bc.Task("Generating typescript bindings")

	doc := bc.NewDoc("./typescript/messages.ts")

	rev, err := exec.Command("git", "rev-parse", "HEAD").CombinedOutput()
	must(err)

	doc.Line("")
	doc.Line("// These bindings were generated by busegen")
	doc.Line("// - At %s", time.Now())
	doc.Line("// - From https://github.com/itchio/butler/commit/%s", rev)
	doc.Line("// See <https://docs.itch.ovh/buse/master> for a human-friendly documentation")

	doc.Line("")
	doc.Line("import { createRequest, createNotification, Client, IRequest, INotification } from %#v;", "./client")

	scope := newScope()
	must(scope.Assimilate("github.com/itchio/butler/buse", "types.go"))
	must(scope.Assimilate("github.com/itchio/go-itchio", "types.go"))
	must(scope.Assimilate("github.com/itchio/butler/configurator", "types.go"))
	must(scope.Assimilate("github.com/itchio/butler/installer/bfs", "receipt.go"))

	bindType := func(entry *Entry) {
		doc.Line("")
		doc.Line("/**")
		switch entry.kind {
		case EntryKindParams:
			doc.Line(" * Params for %s", entry.name)
		case EntryKindResult:
			params := scope.FindEntry(strings.TrimSuffix(entry.typeName, "Result") + "Params")
			doc.Line(" * Result for %s", params.name)
		case EntryKindNotification:
			doc.Line(" * Payload for %s", entry.name)
		default:
			if len(entry.doc) == 0 {
				doc.Line(" * undocumented")
			} else {
				for _, line := range entry.doc {
					doc.Line(" * %s", line)
				}
			}
		}
		doc.Line(" */")
		switch entry.typeKind {
		case EntryTypeKindStruct:
			doc.Line("export interface %s {", entry.typeName)
			if len(entry.structFields) == 0 {
				doc.Line("  // no fields")
			} else {
				for _, sf := range entry.structFields {
					if len(sf.doc) == 0 {
						doc.Line("  /** undocumented */")
					} else if len(sf.doc) == 1 {
						doc.Line("  /** %s */", sf.doc[0])
					} else {
						doc.Line("  /**")
						for _, line := range sf.doc {
							doc.Line("   * %s", line)
						}
						doc.Line("   */")
					}

					var optionalMarker = ""
					if sf.optional {
						optionalMarker = "?"
					}

					doc.Line("  %s%s: %s;", sf.name, optionalMarker, sf.typeString)
				}
			}

			doc.Line("}")
		case EntryTypeKindEnum:
			doc.Line("export enum %s {", entry.typeName)
			for _, val := range entry.enumValues {
				for _, line := range val.doc {
					doc.Line("  // %s", line)
				}
				// special case for "386", woo
				name := val.name
				if strings.ContainsAny(name[0:1], "0123456789") {
					name = "_" + name
				}
				doc.Line("  %s = %s,", name, val.value)
			}
			doc.Line("}")
		}
	}

	for _, category := range scope.categoryList {
		cat := scope.categories[category]
		for _, entry := range cat.entries {
			bindType(entry)

			switch entry.kind {
			case EntryKindResult:
				paramsTypeName := strings.TrimSuffix(entry.typeName, "Result") + "Params"
				resultTypeName := entry.typeName
				params := scope.FindEntry(paramsTypeName)
				method := params.name
				symbolName := strings.Replace(method, ".", "", -1)

				doc.Line("")
				doc.Line("/**")
				if len(params.doc) == 0 {
					doc.Line(" * undocumented")
				} else {
					for _, line := range params.doc {
						doc.Line(" * %s", line)
					}
				}
				doc.Line(" */")
				doc.Line("export const %s = ", symbolName)
				doc.Line("	createRequest<%s, %s>(%#v);", paramsTypeName, resultTypeName, method)
			case EntryKindNotification:
				method := entry.name
				symbolName := strings.Replace(method, ".", "", -1)

				doc.Line("")
				doc.Line("")
				doc.Line("/**")
				if len(entry.doc) == 0 {
					doc.Line(" * undocumented")
				} else {
					for _, line := range entry.doc {
						doc.Line(" * %s", line)
					}
				}
				doc.Line(" */")
				doc.Line("export const %s = ", symbolName)
				doc.Line("	createNotification<%s>(%#v);", entry.typeName, method)
			}
		}
	}

	doc.Commit("")
	doc.Write()

	return nil
}

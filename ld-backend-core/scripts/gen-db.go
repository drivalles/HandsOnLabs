package main

import (
	"ld-backend/internal/spec"

	"gorm.io/gen"
)

func main() {
	generateCoreDbQueries()
	generateTenantDbQueries()
}

func generateCoreDbQueries() {
	g := gen.NewGenerator(gen.Config{
		OutPath: "internal/impl/back/db/core/query", // where the generated code will live
		Mode:    gen.WithoutContext | gen.WithDefaultQuery,
	})

	for _, t := range spec.DbCoreTables {
		g.ApplyBasic(t)
	}
	g.Execute()
}

func generateTenantDbQueries() {
	g := gen.NewGenerator(gen.Config{
		OutPath: "internal/impl/back/db/tenant/query", // where the generated code will live
		Mode:    gen.WithoutContext | gen.WithDefaultQuery,
	})

	g.ApplyBasic(spec.DbPipelineStage{})
	g.ApplyBasic(spec.DbContact{})
	g.ApplyBasic(spec.DbUser{})
	g.ApplyBasic(spec.DbPipeline{})
	g.ApplyBasic(spec.DbCall{})
	g.ApplyBasic(spec.DbContactTag{})
	g.ApplyBasic(spec.DbDid{})
	g.ApplyBasic(spec.DbOpportunity{})
	g.ApplyBasic(spec.DbCacheTime{})
	g.ApplyBasic(spec.DbTenantSettings{})
	g.ApplyBasic(spec.DbDialerSession{})
	g.ApplyBasic(spec.DbTransferContact{})
	g.ApplyBasic(spec.DbDisposition{})
	g.ApplyBasic(spec.DbCallScript{})
	g.ApplyBasic(spec.DbDidHealthReport{})
	g.Execute()
}

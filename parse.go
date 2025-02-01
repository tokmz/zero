package main

// parseRelations 解析表关联关系
func parseRelations(relations []Relation) *Relations {
	result := &Relations{}
	for _, rel := range relations {
		targetModel := ToCamelCase(rel.Target)
		switch rel.Type {
		case "has_many":
			result.HasMany = append(result.HasMany, HasManyRelation{
				Table:      targetModel,
				ForeignKey: rel.ForeignKey,
				References: rel.References,
			})
		case "has_one":
			result.HasOne = append(result.HasOne, HasOneRelation{
				Table:      targetModel,
				ForeignKey: rel.ForeignKey,
				References: rel.References,
			})
		case "belongs_to":
			result.BelongsTo = append(result.BelongsTo, BelongsToRelation{
				Table:      targetModel,
				ForeignKey: rel.ForeignKey,
				References: rel.References,
			})
		case "many2many":
			result.ManyToMany = append(result.ManyToMany, ManyToManyRelation{
				Table:          targetModel,
				JoinTable:      ToCamelCase(rel.JoinTable),
				JoinForeignKey: rel.ForeignKey,
				References:     rel.References,
				JoinReferences: rel.JoinReferences,
			})
		}
	}
	return result
}

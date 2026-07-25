package dice

// DamageResult 伤害计算结果
type DamageResult struct {
	BaseDmg    int64   // 基础伤害（未修正）
	FinalDmg   int64   // 最终伤害（已修正）
	D20        int64   // d20 出目
	CritText   string  // 暴击/大失败文本
	RollPct    int64   // 攻击掷骰百分比
	StabMul    float64 // STAB 倍率
	TypeMod    float64 // 属性克制修正
	EffectText string  // 效果文本（"效果拔群！"等）
	Hit        bool    // 是否命中
	Crit       bool    // 是否暴击
	Attacker   string  // 攻击者名称
	Defender   string  // 防御者名称
	Power      int64   // 招式威力
	AtkType    string  // 攻击属性
	AtkVal     int64   // 攻击值
	DefVal     int64   // 防御值
	BattleLv   int64   // 战斗等级
}

// calculateDamage 计算伤害
// 参数:
//   - ctx: 上下文
//   - power: 招式威力
//   - atkType: 攻击属性（如 "火", "水"）
//   - isSpecial: 是否为特殊攻击（true=特攻/特防，false=物攻/物防）
//   - advantage: "优势" 或 "劣势" 或 ""
//   - ctLimit: 暴击阈值 (2-20)
//   - attacker: 攻击者名称（用于查找 NPC 数据）
//   - defender: 防御者名称（用于查找 NPC 数据）
//
// 返回:
//   - DamageResult: 伤害计算结果
//   - string: 错误信息（空表示无错误）
func calculateDamage(ctx *MsgContext, power int64, atkType string, isSpecial bool,
	advantage string, ctLimit int64, attacker string, defender string) (DamageResult, string) {

	var result DamageResult

	// 攻击者上下文（用于回退）
	attackerCtx := ctx
	defCtx := ctx

	// ----- 1. 获取攻击者攻击值 -----
	atkVal := int64(10)
	if !isSpecial {
		// 物理攻击：读取 patk
		atkVal = getNPCAttr(ctx, attacker, "patk")
		if atkVal == 0 {
			if v, _ := VarGetValueInt64(attackerCtx, "patk"); v != 0 {
				atkVal = v
			} else {
				atkVal = 10
			}
		}
	} else {
		// 特殊攻击：读取 satk
		atkVal = getNPCAttr(ctx, attacker, "satk")
		if atkVal == 0 {
			if v, _ := VarGetValueInt64(attackerCtx, "satk"); v != 0 {
				atkVal = v
			} else {
				atkVal = 10
			}
		}
	}

	// ----- 2. 获取防御者防御值 -----
	defVal := int64(10)
	if !isSpecial {
		// 物理防御：读取 pdef
		defVal = getNPCAttr(ctx, defender, "pdef")
		if defVal == 0 {
			if v, _ := VarGetValueInt64(defCtx, "pdef"); v != 0 {
				defVal = v
			} else {
				defVal = 10
			}
		}
	} else {
		// 特殊防御：读取 sdef
		defVal = getNPCAttr(ctx, defender, "sdef")
		if defVal == 0 {
			if v, _ := VarGetValueInt64(defCtx, "sdef"); v != 0 {
				defVal = v
			} else {
				defVal = 10
			}
		}
	}

	// ----- 3. 获取战斗等级（默认30，符合PMDnD初始CR） -----
	battleLv := int64(30)

	// 优先从攻击者 NPC 数据读取 cr
	if v := getNPCAttr(ctx, attacker, "cr"); v > 0 {
		battleLv = v
	} else if v, _ := VarGetValueInt64(attackerCtx, "战斗等级"); v != 0 {
		battleLv = v
	}

	// ----- 4. 掷骰 (d20) -----
	d20Expr := "d20"
	if advantage != "" {
		d20Expr = "d20" + advantage
	}
	attackerCtx.CreateVmIfNotExists()
	r := attackerCtx.Eval(d20Expr, nil)
	if r.vm.Error != nil {
		return result, "骰点失败: " + r.vm.Error.Error()
	}
	d20, _ := r.ReadInt()
	result.D20 = int64(d20)

	// ----- 5. 计算攻击掷骰百分比 -----
	rollPct := int64(d20) * 5
	if int64(d20) >= ctLimit {
		rollPct += 50
		result.CritText = "【暴击+50%】"
	}
	if int64(d20) == 1 {
		rollPct = 0
		result.CritText = "【大失败】"
	}
	result.RollPct = rollPct

	// ----- 6. 计算基础伤害 -----
	totalDmg := (power * battleLv * atkVal * rollPct) / (100 * defVal)
	if totalDmg < 1 {
		totalDmg = 1
	}
	if rollPct == 0 {
		totalDmg = 0
	}
	result.BaseDmg = totalDmg

	// ----- 7. STAB 计算（本系加成） -----
	atkTypes := []string{}
	for _, t := range pmdndTypeNames {
		key := "type_" + t
		if v := getNPCAttr(ctx, attacker, key); v > 0 {
			atkTypes = append(atkTypes, t)
			continue
		}
		if v, _ := VarGetValueInt64(attackerCtx, key); v > 0 {
			atkTypes = append(atkTypes, t)
		}
	}

	stabMul := 1.0
	for _, t := range atkTypes {
		if t == atkType {
			key := "stab_" + t
			if v := getNPCAttr(ctx, attacker, key); v > 0 {
				stabMul = (100.0 + float64(v)) / 100.0
			} else if v, _ := VarGetValueInt64(attackerCtx, key); v > 0 {
				stabMul = (100.0 + float64(v)) / 100.0
			} else {
				stabMul = 1.5
			}
			break
		}
	}
	result.StabMul = stabMul

	// ----- 8. 属性克制计算（支持多属性叠加） -----
	typeMod := 0.0
	if defender != "" && defender != "目标" {
		for _, t := range pmdndTypeNames {
			key := "type_" + t
			var typeVal int64

			// 检查 NPC
			if v := getNPCAttr(ctx, defender, key); v > 0 {
				typeVal = v
			} else if v, _ := VarGetValueInt64(defCtx, key); v > 0 {
				typeVal = v
			}

			if typeVal > 0 {
				if m, ok := pmdndTypeChart[atkType][t]; ok {
					// 克制系数乘以属性层数（type_火:1 表示一层）
					typeMod += m * float64(typeVal)
				}
			}
		}
	}
	result.TypeMod = typeMod

	// ----- 9. 计算最终伤害（应用 STAB 和克制修正） -----
	finalDmg := totalDmg
	if typeMod != 0 || stabMul != 1.0 {
		factor := (2.0 + typeMod) / 2.0
		if factor < 0.25 {
			factor = 0.25
		}
		factor *= stabMul
		finalDmg = int64(float64(totalDmg) * factor)
	}
	result.FinalDmg = finalDmg

	// ----- 10. 填充结果字段 -----
	result.Attacker = attacker
	result.Defender = defender
	result.Power = power
	result.AtkType = atkType
	result.AtkVal = atkVal
	result.DefVal = defVal
	result.BattleLv = battleLv
	result.Hit = rollPct > 0
	result.Crit = result.CritText != "" && result.CritText != "【大失败】"

	// 计算总修正系数用于效果文本
	totalFactor := 1.0
	if typeMod != 0 || stabMul != 1.0 {
		factor := (2.0 + typeMod) / 2.0
		if factor < 0.25 {
			factor = 0.25
		}
		totalFactor = factor * stabMul
	}

	// 判定效果文本
	if finalDmg == 0 && rollPct != 0 {
		result.EffectText = "对 " + defender + " 没有效果……"
	} else if totalFactor >= 2.0 {
		result.EffectText = "效果拔群！"
	} else if totalFactor <= 0.5 && totalFactor > 0 {
		result.EffectText = "效果不彰……"
	} else {
		result.EffectText = ""
	}

	return result, ""
}

// HealResult 治疗计算结果
type HealResult struct {
	BaseHeal   int64   // 基础治疗量
	FinalHeal  int64   // 最终治疗量
	D20        int64   // d20 出目
	CritText   string  // 暴击文本
	RollPct    int64   // 治疗掷骰百分比
	StabMul    float64 // STAB 倍率
	Crit       bool    // 是否暴击
	HealAtkVal int64   // 治疗攻击值（特防）
}

// calculateHeal 计算治疗量
// 规则: 治疗使用特防作为攻击能力值，防御方能力值固定为200
func calculateHeal(ctx *MsgContext, power int64, atkType string, advantage string, ctLimit int64, attacker string, defender string) (HealResult, string) {
	var result HealResult

	attackerCtx := ctx

	// ----- 1. 获取治疗攻击值（特防） -----
	healAtkVal := getNPCAttr(ctx, attacker, "sdef")
	if healAtkVal == 0 {
		if v, _ := VarGetValueInt64(attackerCtx, "sdef"); v != 0 {
			healAtkVal = v
		} else {
			healAtkVal = 10 // 默认值
		}
	}
	result.HealAtkVal = healAtkVal

	// ----- 2. 治疗公式：挑战等级固定为 100，防御固定为 200 -----
	battleLv := int64(100)
	defVal := int64(200)

	// ----- 3. 掷骰 (d20) -----
	d20Expr := "d20"
	if advantage != "" {
		d20Expr = "d20" + advantage
	}
	attackerCtx.CreateVmIfNotExists()
	r := attackerCtx.Eval(d20Expr, nil)
	if r.vm.Error != nil {
		return result, "骰点失败: " + r.vm.Error.Error()
	}
	d20, _ := r.ReadInt()
	result.D20 = int64(d20)

	rollPct := int64(d20) * 5
	if int64(d20) >= ctLimit {
		rollPct += 50
		result.CritText = "【暴击+50%】"
	}
	if int64(d20) == 1 {
		rollPct = 0
		result.CritText = "【大失败】"
	}
	result.RollPct = rollPct
	result.Crit = int64(d20) >= ctLimit

	// ----- 4. 基础治疗量 -----
	baseHeal := (power * battleLv * healAtkVal * rollPct) / (100 * defVal)
	if baseHeal < 1 {
		baseHeal = 1
	}
	if rollPct == 0 {
		baseHeal = 0
	}
	result.BaseHeal = baseHeal

	// ----- 5. STAB 计算 -----
	atkTypes := []string{}
	for _, t := range pmdndTypeNames {
		key := "type_" + t
		if v := getNPCAttr(ctx, attacker, key); v > 0 {
			atkTypes = append(atkTypes, t)
			continue
		}
		if v, _ := VarGetValueInt64(attackerCtx, key); v > 0 {
			atkTypes = append(atkTypes, t)
		}
	}

	stabMul := 1.0
	for _, t := range atkTypes {
		if t == atkType {
			key := "stab_" + t
			if v := getNPCAttr(ctx, attacker, key); v > 0 {
				stabMul = (100.0 + float64(v)) / 100.0
			} else if v, _ := VarGetValueInt64(attackerCtx, key); v > 0 {
				stabMul = (100.0 + float64(v)) / 100.0
			} else {
				stabMul = 1.5
			}
			break
		}
	}
	result.StabMul = stabMul

	// ----- 6. 最终治疗量 -----
	finalHeal := baseHeal
	if stabMul != 1.0 {
		finalHeal = int64(float64(baseHeal) * stabMul)
	}
	result.FinalHeal = finalHeal

	return result, ""
}

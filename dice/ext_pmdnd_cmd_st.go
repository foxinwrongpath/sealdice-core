package dice

import (
	"fmt"

	ds "github.com/sealdice/dicescript"
)

var cmdSt = getCmdStBase(CmdStOverrideInfo{
	Help: ".st 模板 // 录卡模板\n" +
		".st show // 展示个人属性\n" +
		".st show <属性1> <属性2> ... // 展示特定的属性数值\n" +
		".st show <数字> // 展示高于<数字>的属性\n" +
		".st clr/clear // 清除属性\n" +
		".st del <属性1> <属性2> ... // 删除属性\n" +
		".st export // 导出属性\n" +
		".st help // 帮助\n" +
		".st <属性>:<值> // 设置属性，例：.st 感知:20 洞悉:3\n" +
		".st <属性>±<表达式> // 修改属性，例：.st hp+1d4\n" +
		".st <属性>±<表达式> @某位 // 修改其他宝可梦属性\n",
	TemplateName: "pmdnd",
	CommandSolve: func(ctx *MsgContext, msg *Message, cmdArgs *CmdArgs) *CmdExecuteResult {
		val := cmdArgs.GetArgN(1)
		switch val {
		case "模板":
			text := "PMDnD人物卡模板:\n"
			text += ".dst 力量:10 体质:10 敏捷:10 智力:10 感知:10 魅力:10 hp:10 hpmax:10 pp:10 patk:10 pdef:10 satk:10 sdef:10 spd:10 熟练:2 运动:0 体操:0 巧手:0 隐匿:0 调查:0 奥秘:0 历史:0 自然:0 宗教:0 察觉:0 洞悉:0 驯兽:0 医药:0 求生:0 游说:0 欺瞒:0 威吓:0 表演:0\n"
			text += "PMDnD特有属性: patk(物理攻击) pdef(物理防御) satk(特殊攻击) sdef(特殊防御) spd(速度) pp(技能点)\n"
			text += "战斗资源: 行动力(action) 附加行动(bonusAction) 反应(reaction)\n"
			text += "技能只写修正值，调整值会自动计算。熟练写为\"运动*:0\""
			ReplyToSender(ctx, msg, text)
			return &CmdExecuteResult{Matched: true, Solved: true}
		}
		ctx.CreateVmIfNotExists()
		ctx.setDndReadForVM(false)
		return nil
	},
	ToShow: func(ctx *MsgContext, k string, v *ds.VMValue, tmpl *GameSystemTemplate) string {
		suffixText := ""
		ctx.CreateVmIfNotExists()
		ctx.setDndReadForVM(false)
		orgV, err := ctx.vm.RunExpr("$org_"+k, true)
		if orgV != nil {
			if orgV.TypeId == ds.VMTypeComputedValue {
				return ""
			}
			vOut := orgV.ToString()
			if vOut != v.ToString() {
				suffixText = fmt.Sprintf("[%s]", vOut)
			}
			if err != nil {
				suffixText = fmt.Sprintf("[%s]", err.Error())
			}
		}
		return fmt.Sprintf("%s:%s%s", k, v.ToString(), suffixText)
	},
	ToMod: func(ctx *MsgContext, args *CmdArgs, i *stSetOrModInfoItem, attrs *AttributesItem, tmpl *GameSystemTemplate) bool {
		over := args.GetKwarg("over")
		attrName := tmpl.GetAlias(i.name)
		if attrName == "hp" && over == nil {
			hpBuff := attrs.Load("$buff_hp")
			if hpBuff == nil {
				hpBuff = ds.NewIntVal(0)
			}
			vHpBuffVal := hpBuff.MustReadInt()
			if vHpBuffVal > 0 && i.op == "-" {
				val := vHpBuffVal - i.value.MustReadInt()
				if val >= 0 {
					attrs.Store("$buff_hp", ds.NewIntVal(val))
					i.value = ds.NewIntVal(0)
				} else {
					attrs.Delete("$buff_hp")
					i.value = ds.NewIntVal(-val)
				}
			}
		}

		parent := dndAttrParent[attrName]
		if parent != "" {
			val := attrs.Load(attrName)
			if val == nil {
				m := ds.ValueMap{}
				m.Store("base", ds.NewIntVal(0))
				m.Store("factor", ds.NewIntVal(0))
				val = ds.NewComputedValRaw(&ds.ComputedData{
					Expr:  fmt.Sprintf("pbCalc(this.base, this.factor, %s)", parent),
					Attrs: &m,
				})
				attrs.Store(attrName, val)
			}
			if val.TypeId == ds.VMTypeComputedValue {
				cd, _ := val.ReadComputed()
				base, _ := cd.Attrs.Load("base")
				if base == nil {
					base = ds.NewIntVal(0)
				}
				var vNew *ds.VMValue
				if i.op == "+" {
					vNew = base.OpAdd(ctx.vm, i.value)
				}
				if i.op == "-" {
					vNew = base.OpSub(ctx.vm, i.value)
				}
				if vNew != nil {
					cd.Attrs.Store("base", vNew)
					return true
				}
			}
		}
		return false
	},
	ToModResult: func(ctx *MsgContext, args *CmdArgs, i *stSetOrModInfoItem, attrs *AttributesItem, tmpl *GameSystemTemplate, theOldValue, theNewValue *ds.VMValue) *ds.VMValue {
		attrName := tmpl.GetAlias(i.name)
		if attrName == "hp" {
			var curHpMax ds.IntType
			hpMax, maxExists := attrs.LoadX("hpmax")
			if maxExists {
				switch hpMax.TypeId {
				case ds.VMTypeComputedValue:
					cd, _ := hpMax.ReadComputed()
					curHpMax, _ = ctx.Eval(cd.Expr, nil).ReadInt()
				default:
					curHpMax, _ = hpMax.ReadInt()
				}
			}
			if hpmaxBuff, exits := attrs.LoadX("$buff_hpmax"); exits {
				maxVal, _ := hpmaxBuff.ReadInt()
				curHpMax += maxVal
			}
			newHp, _ := theNewValue.ReadInt()
			if newHp > curHpMax {
				return ds.NewIntVal(curHpMax)
			}
			if theOldValue != nil {
				oldHp, _ := theOldValue.ReadInt()
				if newHp <= 0 && int64(oldHp) > 0 {
					triggerDeathSave(ctx, ctx.Player.Name)
				}
			}
		}
		return theNewValue
	},
	ToSet: func(ctx *MsgContext, i *stSetOrModInfoItem, attrs *AttributesItem, tmpl *GameSystemTemplate) bool {
		attrName := tmpl.GetAlias(i.name)
		parent := dndAttrParent[attrName]
		if parent != "" {
			m := ds.ValueMap{}
			m.Store("base", i.value)
			if i.extra != nil {
				m.Store("factor", i.extra)
			} else {
				m.Delete("factor")
			}
			i.value = ds.NewComputedValRaw(&ds.ComputedData{
				Expr:  fmt.Sprintf("pbCalc(this.base, this.factor, %s)", parent),
				Attrs: &m,
			})
		} else if isAbilityScores(attrName) {
			if i.extra != nil {
				attrs.Store(stpFormat(attrName), i.extra)
			} else {
				attrs.Delete(stpFormat(attrName))
			}
		}
		attrs.Store(attrName, i.value)
		return true
	},
})

package AdminControllers

import (
	"strings"

	"github.com/TruthHun/DocHub/helper"
	"github.com/TruthHun/DocHub/models"
)

type SysController struct {
	BaseController
}

//系统配置管理
func (this *SysController) Get() {
	tab := models.ConfigCate(strings.ToLower(this.GetString("tab")))
	switch tab {
	case models.CONFIG_EMAIL, models.CONFIG_OSS, models.CONFIG_DEPEND, models.CONFIG_ELASTICSEARCH, models.CONFIG_LOGS:
	default:
		tab = "default"
	}
	if this.Ctx.Request.Method == "POST" {
		if tab == "default" {
			var sys models.Sys
			this.ParseForm(&sys)
			if i, err := models.O.Update(&sys); i > 0 && err == nil {
				models.ModelSys.UpdateGlobal() //更新全局变量
			} else {
				if err != nil {
					helper.Logger.Error(err.Error())
				}
				this.ResponseJson(0, "更新失败，可能您未对内容做更改")
			}
		} else {
			modelCfg := new(models.Config)
			for k, v := range this.Ctx.Request.Form {
				modelCfg.UpdateByKey(models.ConfigCate(tab), k, v[0])
			}
			//最后更新全局配置
			modelCfg.UpdateGlobal()
			if tab == models.CONFIG_ELASTICSEARCH {
				if err := models.NewElasticSearchClient().Init(); err != nil {
					this.ResponseJson(0, "ElasticSearch初始化失败："+err.Error())
				}
			}
		}
		this.ResponseJson(1, "更新成功")
	} else {
		this.Data["Tab"] = tab
		this.Data["Title"] = "系统管理"
		this.Data["IsSys"] = true
		if tab == "default" {
			this.Data["Sys"], _ = models.ModelSys.Get()
		} else {
			this.Data["Configs"] = new(models.Config).All()
			if tab == models.CONFIG_ELASTICSEARCH {
				count, errES := models.NewElasticSearchClient().Count()
				this.Data["Count"] = count
				if errES != nil {
					this.Data["ErrES"] = errES.Error()
				}
			}
		}
		this.TplName = "index.html"
	}

}

//重建全量索引
func (this *SysController) RebuildAllIndex() {
	resp := map[string]interface{}{
		"status": 0, "msg": "全量索引重建失败",
	}
	if client := models.NewElasticSearchClient(); client.On {
		//再次初始化，避免elasticsearch未初始化
		if err := client.Init(); err != nil {
			resp["msg"] = "全量索引重建失败：" + err.Error()
		} else {
			//索引是否正在进行，如果正在进行，则不再执行全量索引
			exist := false
			if indexing, ok := helper.GlobalConfigMap.Load("indexing"); ok {
				if b, ok := indexing.(bool); b && ok { //索引正在重建
					exist = true
				} else {
					exist = false
				}
			} else { //不存在正在重建的全量索引操作
				exist = false
			}
			if exist == false {
				go client.RebuildAllIndex()
				resp["msg"] = "全量索引重建提交成功，正在后端执行，请耐心等待."
				resp["status"] = 1
			} else {
				resp["msg"] = "全量索引重建失败：存在正在重建的全量索引"
			}
		}
	} else {
		resp["msg"] = "全量索引重建失败，您未启用ElasticSearch"
	}
	this.Response(resp)
}

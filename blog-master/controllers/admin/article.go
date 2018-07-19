package admin

import (
	"fmt"
	"github.com/astaxie/beego/orm"
	"blog-master/models"
	"os"
	"strconv"
	"strings"
	"time"
    "math/rand"
)

type ArticleController struct {
	baseController
}

//添加
func (this *ArticleController) Add() {
	this.Data["posttime"] = this.getTime().Format("2006-01-02 15:04:05")
	this.display()
}

//管理
func (this *ArticleController) List() {
	var (
		page       int64
		pagesize   int64 = 10
		status     int64
		offset     int64
		list       []*models.Post
		post       models.Post
		searchtype string
		keyword    string
	)
	searchtype = this.GetString("searchtype")
	keyword = this.GetString("keyword")
	status, _ = this.GetInt64("status")
	if page, _ = this.GetInt64("page"); page < 1 {
		page = 1
	}
	offset = (page - 1) * pagesize
	query := post.Query().Filter("status", status).RelatedSel()

	if keyword != "" {
		switch searchtype {
		case "title":
			query = query.Filter("title__icontains", keyword)
		case "author":
			query = query.Filter("user__username__icontains", keyword)
		case "tag":
			query = query.Filter("tags__icontains", keyword)
		}
	}
	count, _ := query.Count()
	if count > 0 {
		query.OrderBy("-istop", "-posttime").Limit(pagesize, offset).RelatedSel().All(&list)
	}

	this.Data["searchtype"] = searchtype
	this.Data["keyword"] = keyword
	this.Data["count_1"], _ = post.Query().Filter("status", 1).Count()
	this.Data["count_2"], _ = post.Query().Filter("status", 2).Count()
	this.Data["status"] = status
	this.Data["list"] = list
	this.Data["pagebar"] = models.NewPager(page, count, pagesize, fmt.Sprintf("/admin/article/list?status=%d&searchtype=%s&keyword=%s&page=%s", status, searchtype, keyword, "%d")).ToString()
	this.display()
}

//编辑
func (this *ArticleController) Edit() {
	id, _ := this.GetInt64("id")
	post := models.Post{Id: id}
	if post.Read() != nil {
		this.Abort("404")
	}
	post.Tags = strings.Trim(post.Tags, ",")
	this.Data["post"] = post
	this.Data["posttime"] = post.Posttime.Format("2006-01-02 15:04:05")
	this.display()
}

//保存
func (this *ArticleController) Save() {
	var (
		id      int64  = 0
		title   string = strings.TrimSpace(this.GetString("title"))
		content string = this.GetString("content")
		tags    string = strings.TrimSpace(this.GetString("tags"))
		urlname string = strings.TrimSpace(this.GetString("urlname"))
		color   string = strings.TrimSpace(this.GetString("color"))
		cover   string = strings.TrimSpace(this.GetString("cover"))
		timestr string = strings.TrimSpace(this.GetString("posttime"))
		status  int64  = 0
		istop   int8   = 0
		urltype int8   = 0
		post    models.Post
	)
	if title == "" {
		this.showmsg("标题不能为空！")
	}

	id, _ = this.GetInt64("id")
	status, _ = this.GetInt64("status")

	if this.GetString("istop") == "1" {
		istop = 1
	}
	if this.GetString("urltype") == "1" {
		urltype = 1
	}
	if status != 1 && status != 2 {
		status = 0
	}
	if cover == "" {
		cover = fmt.Sprintf("/static/upload/default/blog-default-%d.png", rand.Intn(9))
	}
	addtags := make([]string, 0)
	//标签过滤
	if tags != "" {
		tagarr := strings.Split(tags, ",")
		for _, v := range tagarr {
			if tag := strings.TrimSpace(v); tag != "" {
				exists := false
				for _, vv := range addtags {
					if vv == tag {
						exists = true
						break
					}
				}
				if !exists {
					addtags = append(addtags, tag)
				}
			}
		}
	}

	if id < 1 {
		post.User = &models.User{Id:this.userid}
		post.Posttime = this.getTime()
		post.Updated = this.getTime()
		post.Insert()
		models.Cache.Delete("latestblog")
	} else {
		post.Id = id
		if post.Read() != nil {
			goto RD
		}
		lastcover := post.Cover
		if lastcover != cover && lastcover != "" && lastcover != "/static/upload/default/blog-default-0.png" {
			os.Remove("."+lastcover)
		}
		if post.Tags != "" {
			var tagobj models.Tag
			var tagpostobj models.TagPost
			oldtags := strings.Split(strings.Trim(post.Tags, ","), ",")
			//标签统计-1
			tagobj.Query().Filter("name__in", oldtags).Update(orm.Params{"count": orm.ColValue(orm.ColMinus, 1)})
			//删掉tag_post表的记录
			tagpostobj.Query().Filter("postid", post.Id).Delete()
		}
	}

	if len(addtags) > 0 {
		for _, v := range addtags {
			tag := models.Tag{Name: v}
			if tag.Read("Name") == orm.ErrNoRows {
				tag.Count = 1
				tag.Insert()
			} else {
				tag.Count += 1
				tag.Update("Count")
			}
			tp := models.TagPost{Tagid: tag.Id, Postid: post.Id, Poststatus: int8(status), Posttime: this.getTime()}
			tp.Insert()
		}
		post.Tags = "," + strings.Join(addtags, ",") + ","
	}
	if posttime, err := time.Parse("2006-01-02 15:04:05", timestr); err == nil {
		post.Posttime = posttime
	} else {
		post.Posttime, _ = time.Parse("2006-01-02 15:04:05", post.Posttime.Format("2006-01-02 15:04:05"))
	}
	post.Status = int8(status)
	post.Title = title
	post.Color = color
	post.Istop = istop
	post.Cover = cover
	post.Content = content
	post.Urlname = urlname
	post.Urltype = urltype
	post.Updated = this.getTime()
	post.Update("tags", "status", "title", "color", "cover", "istop", "content", "urlname", "urltype", "updated", "posttime")

RD:
	this.Redirect("/admin/article/list", 302)
}

//删除
func (this *ArticleController) Delete() {
	id, _ := this.GetInt64("id")
	post := models.Post{Id: id}
	if post.Read() == nil {
		if post.Cover != "" && post.Cover != "/static/upload/default/blog-default-0.png" {
			os.Remove("."+post.Cover)
		}
		post.Delete()
	}
	models.Cache.Delete("latestblog")
	this.Redirect("/admin/article/list", 302)
}

//批处理
func (this *ArticleController) Batch() {
	ids := this.GetStrings("ids[]")
	op := this.GetString("op")

	idarr := make([]int64, 0)
	for _, v := range ids {
		if id, _ := strconv.Atoi(v); id > 0 {
			idarr = append(idarr, int64(id))
		}
	}

	var post models.Post

	switch op {
	case "topub": //移到已发布
		post.Query().Filter("id__in", idarr).Update(orm.Params{"status": 0})
	case "todrafts": //移到草稿箱
		post.Query().Filter("id__in", idarr).Update(orm.Params{"status": 1})
	case "totrash": //移到回收站
		post.Query().Filter("id__in", idarr).Update(orm.Params{"status": 2})
	case "delete": //批量删除
		for _, id := range idarr {
			obj := models.Post{Id: id}
			if obj.Read() == nil {
				obj.Delete()
			}
		}
	}

	this.Redirect(this.Ctx.Request.Referer(), 302)
}



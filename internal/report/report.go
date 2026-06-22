package report

import (
	"sort"

	"github.com/xZhad/jsonldb"
	"github.com/xZhad/pomo/internal/model"
	"github.com/xZhad/pomo/internal/store"
)

func Last(s *store.Store) (model.Session, bool, error) {
	c, err := jsonldb.Open(s.SessionsPath())
	if err != nil {
		return model.Session{}, false, err
	}
	defer c.Close()
	all, err := jsonldb.Typed[model.Session](c).All()
	if err != nil {
		return model.Session{}, false, err
	}
	if len(all) == 0 {
		return model.Session{}, false, nil
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].Started.After(all[j].Started)
	})
	return all[0], true, nil
}

func Log(s *store.Store, filter string) ([]model.Session, error) {
	c, err := jsonldb.Open(s.SessionsPath())
	if err != nil {
		return nil, err
	}
	defer c.Close()
	db := jsonldb.Typed[model.Session](c)
	res, err := db.Query(filter)
	if err != nil {
		return nil, err
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].Started.After(res[j].Started)
	})
	return res, nil
}

type Bucket struct {
	Key          string
	Count        int
	TotalSeconds int
}

func Topics(s *store.Store) ([]Bucket, error) { return Report(s, "topic") }

func Report(s *store.Store, by string) ([]Bucket, error) {
	c, err := jsonldb.Open(s.SessionsPath())
	if err != nil {
		return nil, err
	}
	defer c.Close()
	res, err := c.Query("")
	if err != nil {
		return nil, err
	}

	var buckets []Bucket
	switch by {
	case "tag":
		tagged := map[string]*tagAcc{}
		for _, d := range res.Docs() {
			secs, _ := d.GetFloat("duration")
			tags := tagsOf(d)
			for _, tg := range tags {
				a := tagged[tg]
				if a == nil {
					a = &tagAcc{}
					tagged[tg] = a
				}
				a.count++
				a.secs += int(secs)
			}
		}
		for k, a := range tagged {
			buckets = append(buckets, Bucket{Key: k, Count: a.count, TotalSeconds: a.secs})
		}
	default:
		var groups map[string]*jsonldb.Result
		if by == "day" {
			groups = res.GroupByFunc(func(d jsonldb.Doc) string {
				s := d.GetString("started")
				if len(s) >= 10 {
					return s[:10] // YYYY-MM-DD
				}
				return s
			})
		} else { // "topic"
			groups = res.GroupBy("topic")
		}
		for k, g := range groups {
			total, _ := g.Sum("duration")
			buckets = append(buckets, Bucket{Key: k, Count: g.Count(), TotalSeconds: int(total)})
		}
	}
	sort.Slice(buckets, func(i, j int) bool {
		if buckets[i].TotalSeconds != buckets[j].TotalSeconds {
			return buckets[i].TotalSeconds > buckets[j].TotalSeconds
		}
		return buckets[i].Key < buckets[j].Key
	})
	return buckets, nil
}

type tagAcc struct {
	count int
	secs  int
}

func tagsOf(d jsonldb.Doc) []string {
	v, ok := d.Get("tags")
	if !ok {
		return nil
	}
	arr, ok := v.([]any)
	if !ok {
		return nil
	}
	var out []string
	for _, x := range arr {
		if s, ok := x.(string); ok {
			out = append(out, s)
		}
	}
	return out
}


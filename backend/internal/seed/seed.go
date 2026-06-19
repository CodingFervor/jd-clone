package seed

import (
	"database/sql"
	"log"

	"github.com/CodingFervor/jd-clone/backend/internal/model"
	"github.com/CodingFervor/jd-clone/backend/internal/repository"
)

// Run populates the database with mock data if it is empty. It is idempotent —
// safe to call on every startup.
func Run(db *sql.DB) {
	userRepo := repository.NewUserRepo(db)
	catRepo := repository.NewCategoryRepo(db)
	prodRepo := repository.NewProductRepo(db)
	reviewRepo := repository.NewReviewRepo(db)

	// Seed users if none exist.
	if u, _ := userRepo.FindByUsername("admin"); u == nil && !userRepo.Exists("admin") {
		if err := userRepo.Create(&model.User{
			Username: "admin", Password: "admin123", Nickname: "京东用户", Avatar: "https://img12.360buyimg.com/img/s100x100_jfs/t1/avatar",
		}); err != nil {
			log.Printf("seed user: %v", err)
		}
		if err := userRepo.Create(&model.User{
			Username: "jduser", Password: "123456", Nickname: "数码达人", Avatar: "https://img12.360buyimg.com/img/s100x100_jfs/t2/avatar",
		}); err != nil {
			log.Printf("seed user2: %v", err)
		}
	}

	// Seed categories if none exist.
	n, _ := catRepo.Count()
	if n == 0 {
		cats := []model.Category{
			{Name: "手机数码", Icon: "📱", SortOrder: 1},
			{Name: "家用电器", Icon: "🔌", SortOrder: 2},
			{Name: "电脑办公", Icon: "💻", SortOrder: 3},
			{Name: "服饰鞋包", Icon: "👗", SortOrder: 4},
			{Name: "食品生鲜", Icon: "🍎", SortOrder: 5},
			{Name: "美妆个护", Icon: "💄", SortOrder: 6},
			{Name: "母婴玩具", Icon: "🧸", SortOrder: 7},
			{Name: "家居家装", Icon: "🛋️", SortOrder: 8},
		}
		for _, c := range cats {
			_, _ = db.Exec(`INSERT INTO categories (name, icon, sort_order) VALUES (?,?,?)`, c.Name, c.Icon, c.SortOrder)
		}
	}

	// Seed products if none exist.
	var prodCount int
	_ = db.QueryRow(`SELECT COUNT(*) FROM products`).Scan(&prodCount)
	if prodCount == 0 {
		products := []model.ProductInput{
			// 手机数码 (category_id=1)
			{Name: "Apple iPhone 15 Pro Max 256GB 原色钛金属 5G手机", Subtitle: "A17 Pro芯片 4800万像素 京东自营", Price: 9999, OriginalPrice: 10999, Image: "https://img14.360buyimg.com/n1/jfs/t1/iphone15promax.jpg", Category: "手机数码", CategoryID: 1, Shop: " Apple（苹果）京东自营官方旗舰店", Stock: 500, Sales: 50000, Description: "iPhone 15 Pro Max，钛金属设计，A17 Pro 芯片，专业级摄像头系统。", Tags: "新品,自营,顺丰", IsSeckill: 1},
			{Name: "华为 Mate 60 Pro 12+512GB 雅丹黑", Subtitle: "卫星通话 麒麟9000s 京东自营", Price: 6999, OriginalPrice: 7299, Image: "https://img14.360buyimg.com/n1/jfs/t1/mate60pro.jpg", Category: "手机数码", CategoryID: 1, Shop: "华为京东自营官方旗舰店", Stock: 300, Sales: 80000, Description: "华为 Mate 60 Pro，卫星通话，麒麟9000s芯片，超可靠玄武架构。", Tags: "热门,自营", IsSeckill: 1},
			{Name: "小米 14 Ultra 16+512GB 黑色", Subtitle: "徕卡光学 骁龙8Gen3 摄影旗舰", Price: 6499, OriginalPrice: 6999, Image: "https://img14.360buyimg.com/n1/jfs/t1/mi14ultra.jpg", Category: "手机数码", CategoryID: 1, Shop: "小米京东自营旗舰店", Stock: 200, Sales: 35000, Description: "小米14 Ultra，徕卡专业光学，全焦段大光圈，骁龙8 Gen 3。", Tags: "热门,自营"},
			{Name: "AirPods Pro (第二代) USB-C 接口", Subtitle: "主动降噪 空间音频", Price: 1599, OriginalPrice: 1899, Image: "https://img14.360buyimg.com/n1/jfs/t1/airpodspro2.jpg", Category: "手机数码", CategoryID: 1, Shop: "Apple京东自营", Stock: 800, Sales: 120000, Description: "AirPods Pro 第二代，升级主动降噪，自适应通透模式。", Tags: "爆款", IsSeckill: 1},
			// 家用电器 (category_id=2)
			{Name: "美的变频空调 1.5匹新一级能效挂机", Subtitle: "冷暖两用 静音节能", Price: 2599, OriginalPrice: 3299, Image: "https://img14.360buyimg.com/n1/jfs/t1/midea-ac.jpg", Category: "家用电器", CategoryID: 2, Shop: "美的京东自营旗舰店", Stock: 150, Sales: 20000, Description: "美的变频空调，新一级能效，快速冷暖，低噪运行。"},
			{Name: "海尔冰箱 470升风冷无霜对开门", Subtitle: "一级能效 变频节能", Price: 3199, OriginalPrice: 3999, Image: "https://img14.360buyimg.com/n1/jfs/t1/haier-fridge.jpg", Category: "家用电器", CategoryID: 2, Shop: "海尔京东自营旗舰店", Stock: 100, Sales: 15000, Description: "海尔对开门冰箱，风冷无霜，470升大容量。"},
			// 电脑办公 (category_id=3)
			{Name: "Apple MacBook Pro 14英寸 M3 Pro芯片", Subtitle: "Liquid视网膜XDR显示屏", Price: 14999, OriginalPrice: 15999, Image: "https://img14.360buyimg.com/n1/jfs/t1/macbookpro14.jpg", Category: "电脑办公", CategoryID: 3, Shop: "Apple京东自营", Stock: 80, Sales: 8000, Description: "MacBook Pro 14，M3 Pro芯片，专业性能，超长续航。"},
			{Name: "联想拯救者 Y9000P 16英寸游戏本", Subtitle: "i9-14900HX RTX4070 165Hz", Price: 9999, OriginalPrice: 10999, Image: "https://img14.360buyimg.com/n1/jfs/t1/legion-y9000p.jpg", Category: "电脑办公", CategoryID: 3, Shop: "联想京东自营旗舰店", Stock: 120, Sales: 12000, Description: "联想拯救者 Y9000P，顶级游戏性能，2.5K 165Hz高刷屏。", IsSeckill: 1},
			// 服饰鞋包 (category_id=4)
			{Name: "优衣库 男装 感温衬衫 长袖", Subtitle: "免烫抗皱 商务休闲", Price: 199, OriginalPrice: 299, Image: "https://img14.360buyimg.com/n1/jfs/t1/uniqlo-shirt.jpg", Category: "服饰鞋包", CategoryID: 4, Shop: "优衣库京东自营", Stock: 1000, Sales: 50000},
			{Name: "Nike Air Force 1 经典低帮板鞋", Subtitle: "男女同款 经典百搭", Price: 699, OriginalPrice: 899, Image: "https://img14.360buyimg.com/n1/jfs/t1/nike-af1.jpg", Category: "服饰鞋包", CategoryID: 4, Shop: "Nike京东自营", Stock: 600, Sales: 90000, Description: "Nike Air Force 1，经典低帮，舒适耐穿。", IsSeckill: 1},
			// 食品生鲜 (category_id=5)
			{Name: "农夫山泉 饮用天然水 550ml*24瓶", Subtitle: "整箱装 京东超市", Price: 36, OriginalPrice: 48, Image: "https://img14.360buyimg.com/n1/jfs/t1/nongfu.jpg", Category: "食品生鲜", CategoryID: 5, Shop: "农夫山泉京东自营", Stock: 5000, Sales: 200000},
			{Name: "三只松鼠每日坚果750g混合干果零食", Subtitle: "30小包 京东自营", Price: 89, OriginalPrice: 129, Image: "https://img14.360buyimg.com/n1/jfs/t1/songshu-nuts.jpg", Category: "食品生鲜", CategoryID: 5, Shop: "三只松鼠京东自营旗舰店", Stock: 2000, Sales: 100000, IsSeckill: 1},
			// 美妆个护 (category_id=6)
			{Name: "雅诗兰黛小棕瓶精华50ml 第七代", Subtitle: "抗皱紧致 京东国际", Price: 1080, OriginalPrice: 1380, Image: "https://img14.360buyimg.com/n1/jfs/t1/estee-serum.jpg", Category: "美妆个护", CategoryID: 6, Shop: "雅诗兰黛京东自营", Stock: 200, Sales: 30000},
			{Name: "SK-II 神仙水230ml 精华露", Subtitle: "提亮肤色 紧致毛孔", Price: 1540, OriginalPrice: 1690, Image: "https://img14.360buyimg.com/n1/jfs/t1/sk2.jpg", Category: "美妆个护", CategoryID: 6, Shop: "SK-II京东自营", Stock: 150, Sales: 18000, IsSeckill: 1},
			// 母婴玩具 (category_id=7)
			{Name: "乐高 创意百变 10307 埃菲尔铁塔", Subtitle: "成人收藏级拼装模型", Price: 6999, OriginalPrice: 7999, Image: "https://img14.360buyimg.com/n1/jfs/t1/lego-eiffel.jpg", Category: "母婴玩具", CategoryID: 7, Shop: "乐高京东自营", Stock: 50, Sales: 3000},
			// 家居家装 (category_id=8)
			{Name: "宜家风格 北欧布艺沙发 三人位", Subtitle: "可拆洗 客厅小户型", Price: 1899, OriginalPrice: 2599, Image: "https://img14.360buyimg.com/n1/jfs/t1/sofa.jpg", Category: "家居家装", CategoryID: 8, Shop: "京东家具自营", Stock: 80, Sales: 5000},
		}
		for i := range products {
			if _, err := prodRepo.Create(&products[i]); err != nil {
				log.Printf("seed product %d: %v", i, err)
			}
		}
	}

	// Seed a few reviews on the first product.
	var revCount int
	_ = db.QueryRow(`SELECT COUNT(*) FROM reviews`).Scan(&revCount)
	if revCount == 0 {
		reviews := []model.Review{
			{ProductID: 1, UserID: 2, Username: "数码达人", Rating: 5, Content: "iPhone 15 Pro Max 钛金属质感太棒了，拍照提升明显，京东物流次日达！"},
			{ProductID: 1, UserID: 1, Username: "京东用户", Rating: 5, Content: "包装完好，正品无疑，A17 Pro 性能强劲。"},
			{ProductID: 2, UserID: 2, Username: "数码达人", Rating: 5, Content: "华为 Mate 60 Pro 卫星通话太实用了，信号好，麒麟芯片回归！"},
			{ProductID: 3, UserID: 1, Username: "京东用户", Rating: 4, Content: "徕卡影像名副其实，就是价格略高。"},
		}
		for _, rv := range reviews {
			_ = reviewRepo.Create(&rv)
		}
	}

	log.Println("seed: mock data ensured")
}

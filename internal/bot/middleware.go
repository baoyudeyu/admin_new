package bot

import (
	"admin-bot/internal/cache"
	"admin-bot/internal/config"
	"admin-bot/internal/service"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

// Middleware 中间件接口
type Middleware interface {
	Check(message *tgbotapi.Message) bool
}

// PermissionChecker 权限检查器
type PermissionChecker struct {
	cfg          *config.Config
	adminService *service.AdminService
	groupService *service.GroupService
	bot          *tgbotapi.BotAPI
}

// NewPermissionChecker 创建权限检查器
func NewPermissionChecker(cfg *config.Config, adminService *service.AdminService,
	groupService *service.GroupService, bot *tgbotapi.BotAPI) *PermissionChecker {
	return &PermissionChecker{
		cfg:          cfg,
		adminService: adminService,
		groupService: groupService,
		bot:          bot,
	}
}

// CheckPermission 检查用户权限
func (p *PermissionChecker) CheckPermission(message *tgbotapi.Message) (bool, string) {
	userID := message.From.ID
	chatID := message.Chat.ID

	// 1. 检查是否为作者
	if p.cfg.Telegram.IsAuthor(userID) {
		logrus.Debugf("User %d is author, permission granted", userID)
		return true, "author"
	}

	// 2. 检查是否为全局管理员
	isGlobalAdmin, err := p.adminService.IsGlobalAdmin(userID)
	if err != nil {
		logrus.Errorf("Failed to check global admin: %v", err)
	} else if isGlobalAdmin {
		logrus.Debugf("User %d is global admin, permission granted", userID)
		return true, "global_admin"
	}

	// 3. 检查群组是否已授权
	if !message.Chat.IsGroup() && !message.Chat.IsSuperGroup() {
		// 私聊消息，只有作者和全局管理员可以使用
		logrus.Debugf("Private chat, only author and global admin allowed")
		return false, "private_chat_not_allowed"
	}

	// 检查群组授权
	isAuthorized, err := p.groupService.IsAuthorized(chatID)
	if err != nil {
		logrus.Errorf("Failed to check group authorization: %v", err)
		return false, "check_error"
	}

	if !isAuthorized {
		logrus.Debugf("Group %d is not authorized", chatID)
		return false, "group_not_authorized"
	}

	// 4. 检查是否为群管理员
	if !p.cfg.System.AdminEnabled {
		logrus.Debugf("Admin permission is disabled")
		return false, "admin_disabled"
	}

	// 检查是否为群管理员
	chatMember, err := p.bot.GetChatMember(tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
			ChatID: chatID,
			UserID: userID,
		},
	})

	if err != nil {
		logrus.Errorf("Failed to get chat member: %v", err)
		return false, "check_error"
	}

	// 检查管理员状态
	if chatMember.Status == "creator" || chatMember.Status == "administrator" {
		logrus.Debugf("User %d is group admin, permission granted", userID)
		return true, "group_admin"
	}

	logrus.Debugf("User %d has no permission", userID)
	return false, "no_permission"
}

// IsAuthor 检查是否为作者
func (p *PermissionChecker) IsAuthor(userID int64) bool {
	return p.cfg.Telegram.IsAuthor(userID)
}

// IsGroupAuthorized 检查群组是否已授权（优化版：数据库故障时使用宽容策略）
func (p *PermissionChecker) IsGroupAuthorized(chatID int64) bool {
	// 1. 特殊保护：通知频道永远返回已授权
	authCache := cache.GetAuthCache()
	if authCache.IsNotificationChannel(chatID) {
		logrus.WithField("频道ID", chatID).Debug("✅ 通知频道，跳过授权检查")
		return true
	}

	// 2. 尝试从缓存获取授权状态
	authorized, cached := authCache.IsGroupAuthorized(chatID)
	if cached {
		// 缓存命中，直接返回
		return authorized
	}

	// 3. 缓存未命中，查询数据库
	isAuthorized, err := p.groupService.IsAuthorized(chatID)
	if err != nil {
		// 数据库查询失败 - 采用宽容策略
		logrus.WithFields(logrus.Fields{
			"群组ID": chatID,
			"错误":   err.Error(),
		}).Error("❌ 数据库查询失败，采用宽容策略（保留群组不退出）")

		// 关键优化：数据库故障时，假设群组已授权（避免误退群）
		// 理由：授权后很少删除，数据库故障是暂时的，保守策略更安全
		return true
	}

	return isAuthorized
}

// IsGlobalAdmin 检查是否为全局管理员
func (p *PermissionChecker) IsGlobalAdmin(userID int64) bool {
	isAdmin, err := p.adminService.IsGlobalAdmin(userID)
	if err != nil {
		logrus.Errorf("Failed to check global admin: %v", err)
		return false
	}
	return isAdmin
}

// CheckUserBanned 检查用户是否被拉黑
func CheckUserBanned(userID int64, banService *service.BanService) bool {
	banned, _, err := banService.IsUserBanned(userID)
	if err != nil {
		logrus.Errorf("Failed to check user ban status: %v", err)
		return false
	}
	return banned
}

// CheckUserMuted 检查用户是否被禁言
func CheckUserMuted(userID int64, muteService *service.MuteService) bool {
	muted, _, err := muteService.IsUserMuted(userID)
	if err != nil {
		logrus.Errorf("Failed to check user mute status: %v", err)
		return false
	}
	return muted
}

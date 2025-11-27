package bot

import (
	"admin-bot/internal/service"
	"admin-bot/internal/utils"
	"fmt"
	"regexp"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// CommandParams 命令参数
type CommandParams struct {
	TargetUsers []int64 // 目标用户ID列表
	Duration    int     // 时长（秒）
	Reason      string  // 理由
	IsBatch     bool    // 是否为批量操作
}

// ParseCommand 解析命令
func ParseCommand(message *tgbotapi.Message, bot *tgbotapi.BotAPI, userCacheService *service.UserCacheService) (*CommandParams, error) {
	params := &CommandParams{
		TargetUsers: make([]int64, 0),
		IsBatch:     false,
	}

	// 1. 检查是否有引用回复
	if message.ReplyToMessage != nil {
		params.TargetUsers = append(params.TargetUsers, message.ReplyToMessage.From.ID)
		// 引用回复不算批量操作
		params.IsBatch = false
	}

	// 2. 解析命令参数
	args := strings.Fields(message.Text)
	if len(args) <= 1 {
		// 只有命令本身，没有其他参数
		if len(params.TargetUsers) == 0 {
			return nil, fmt.Errorf("请指定目标用户（引用回复或@用户名）")
		}
		return params, nil
	}

	// 3. 解析参数
	remainingArgs := args[1:]
	var timeStr string
	var reasonParts []string
	userCount := 0

	for _, arg := range remainingArgs {
		if strings.HasPrefix(arg, "@") {
			// 用户名
			username := strings.TrimPrefix(arg, "@")

			// 优先从缓存中查询
			userID, err := userCacheService.GetUserIDByUsername(username)
			if err != nil {
				// 缓存中没有，尝试通过API获取
				userID, err = GetUserIDByUsername(bot, message.Chat.ID, username)
				if err != nil {
					return nil, fmt.Errorf("暂无 @%s 的用户信息，请尝试使用引用回复", username)
				}
			}

			params.TargetUsers = append(params.TargetUsers, userID)
			userCount++
		} else if isDurationString(arg) {
			// 时间
			timeStr = arg
		} else {
			// 理由
			reasonParts = append(reasonParts, arg)
		}
	}

	// 判断是否为批量操作：只有通过@username指定多个用户才算批量
	if userCount > 1 {
		params.IsBatch = true
	}

	// 4. 解析时长
	if timeStr != "" {
		duration, err := utils.ParseDuration(timeStr)
		if err != nil {
			return nil, fmt.Errorf("时间格式错误: %v", err)
		}
		params.Duration = duration
	}

	// 5. 组合理由
	if len(reasonParts) > 0 {
		params.Reason = strings.Join(reasonParts, " ")
	}

	// 6. 检查是否有目标用户
	if len(params.TargetUsers) == 0 {
		return nil, fmt.Errorf("请指定目标用户（引用回复或@用户名）")
	}

	return params, nil
}

// isDurationString 判断是否为时间字符串
func isDurationString(s string) bool {
	matched, _ := regexp.MatchString(`^\d+[smhd]$`, strings.ToLower(s))
	return matched
}

// GetUserIDByUsername 通过用户名获取用户ID
func GetUserIDByUsername(bot *tgbotapi.BotAPI, chatID int64, username string) (int64, error) {
	// 尝试获取聊天成员信息
	member, err := bot.GetChatMember(tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
			ChatID: chatID,
			UserID: 0, // 这里需要用户ID，但我们只有用户名
		},
	})

	if err != nil {
		return 0, err
	}

	return member.User.ID, nil
}

// GetUserInfo 获取用户信息
func GetUserInfo(user *tgbotapi.User) (username, fullName string) {
	username = user.UserName
	fullName = user.FirstName
	if user.LastName != "" {
		fullName += " " + user.LastName
	}
	return
}

// GetChatTitle 获取聊天标题
func GetChatTitle(chat *tgbotapi.Chat) string {
	if chat.Title != "" {
		return chat.Title
	}
	if chat.FirstName != "" {
		return chat.FirstName
	}
	return "未知群组"
}

// GetChatUsername 获取聊天用户名（公开群组/频道）
func GetChatUsername(chat *tgbotapi.Chat) string {
	return chat.UserName
}

// ExtractUserFromMessage 从消息中提取用户信息
func ExtractUserFromMessage(message *tgbotapi.Message) (userID int64, username, fullName string) {
	if message.ReplyToMessage != nil {
		user := message.ReplyToMessage.From
		username, fullName = GetUserInfo(user)
		return user.ID, username, fullName
	}
	return 0, "", ""
}

// ExtractUsersFromEntities 从消息实体中提取用户
func ExtractUsersFromEntities(message *tgbotapi.Message, bot *tgbotapi.BotAPI) ([]int64, error) {
	userIDs := make([]int64, 0)

	if message.Entities == nil || len(message.Entities) == 0 {
		return userIDs, nil
	}

	for _, entity := range message.Entities {
		if entity.Type == "mention" {
			// @username 格式
			start := entity.Offset
			length := entity.Length
			username := message.Text[start+1 : start+length] // 跳过@符号

			userID, err := GetUserIDByUsername(bot, message.Chat.ID, username)
			if err != nil {
				return nil, err
			}
			userIDs = append(userIDs, userID)
		} else if entity.Type == "text_mention" {
			// 文本提及（对于没有用户名的用户）
			if entity.User != nil {
				userIDs = append(userIDs, entity.User.ID)
			}
		}
	}

	return userIDs, nil
}

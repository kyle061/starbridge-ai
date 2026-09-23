package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

func normalizeLoginAgreementMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "checkbox":
		return "checkbox"
	default:
		return defaultLoginAgreementMode
	}
}

func defaultLoginAgreementDocuments() []LoginAgreementDocument {
	return []LoginAgreementDocument{
		{
			ID:        "terms",
			Title:     "Starbridge AI 服务条款",
			ContentMD: defaultTermsOfServiceMarkdown,
		},
		{
			ID:        "privacy-policy",
			Title:     "Starbridge AI 隐私政策",
			ContentMD: defaultPrivacyPolicyMarkdown,
		},
		{
			ID:        "usage-policy",
			Title:     "Starbridge AI 使用政策",
			ContentMD: defaultUsagePolicyMarkdown,
		},
		{
			ID:        "supported-regions",
			Title:     "Starbridge AI 支持的国家和地区",
			ContentMD: defaultSupportedRegionsMarkdown,
		},
		{
			ID:        "service-specific-terms",
			Title:     "Starbridge AI 服务特定条款",
			ContentMD: defaultServiceSpecificTermsMarkdown,
		},
	}
}

// The built-in documents are intentionally conservative templates. Starbridge AI
// is deployed and operated by the site owner, so the owner should review these
// texts for the actual entity, region, retention period, payment flow, and
// contact channel before enabling mandatory consent.
const defaultTermsOfServiceMarkdown = `## 1. 服务说明

Starbridge AI 是由站点运营者部署的多模型 API 网关。站点运营者可以接入 OpenAI 兼容接口、第三方模型服务或自有账号，并根据实际配置提供 API、订阅和用量服务。模型能力、可用地区、响应速度和价格以站点当前展示及实际返回为准。

本模板仅用于帮助站点完成初始配置，不代表开源项目、代码贡献者或上游模型服务商对当前站点提供运营、担保或授权。运营者应在上线前补充主体名称、联系渠道、适用法律和争议解决方式。

## 2. 账号与凭据

- 用户应提供真实、可联系且属于自己的注册信息，并妥善保管密码、API Key、登录令牌和导入配置。
- API Key 仅限本人或获得授权的团队使用。因用户泄露凭据造成的请求、费用或数据损失，由用户及时通知运营者并承担相应责任。
- 运营者可以为账号设置余额、订阅、模型范围、速率和并发限制；具体额度以账户页面和请求响应为准。

## 3. 计费与用量

请求可能按照模型、输入输出 Token、媒体数量、上游实际成本及站点配置的计费倍率扣减余额或订阅额度。请求失败、重试、缓存、异步任务和第三方平台计费可能有不同规则，站点应在价格页面说明实际口径。

当余额、订阅额度或上游账号额度不足时，系统可以拒绝请求、切换到其他可用账号或返回额度不足错误。已产生且可核验的上游费用不因用户删除 Key、取消订阅或关闭账号而自动消失。

## 4. 合法使用

用户不得利用服务实施违法活动、绕过平台安全或地区限制、窃取或滥用他人凭据、发送恶意流量、干扰服务稳定性，或生成和传播违反适用法律及上游平台规则的内容。运营者可以根据风险规则暂停请求、冻结账号或保留必要的审计记录。

## 5. 服务变更与责任

模型和上游接口可能随时调整或中断。运营者会在可行范围内维护服务，但不保证任何模型持续可用、响应无误或满足特定业务结果。因网络、上游平台、配置、不可抗力或用户自身原因导致的中断，运营者在适用法律允许的范围内承担责任。

## 6. 条款更新

运营者可以根据服务、法律或上游规则更新本条款。更新日期和版本以登录页展示为准；如果启用了强制确认，用户需要重新阅读并同意后继续使用。

> 请站点运营者在正式上线前补充主体信息、退款规则、客服微信和适用法律，并让专业人士审阅本模板。`

const defaultPrivacyPolicyMarkdown = `## 1. 适用范围

本隐私政策适用于当前 Starbridge AI 站点的注册、登录、API 调用、订阅、支付和客服流程。站点运营者应根据实际部署地址、运营主体和适用法律补充或修改本政策。

## 2. 我们可能处理的信息

- 账号资料，例如邮箱、用户名、登录方式、认证来源和安全设置；
- 服务运行数据，例如 API Key 的标识、请求时间、模型名称、Token 数量、费用、状态码、延迟和风控结果；
- 设备与网络信息，例如 IP、User-Agent、会话标识和安全审计记录；
- 订阅、余额、订单和退款状态；
- 为排查故障而产生的错误信息和必要的上游响应元数据。

站点运营者应明确说明是否保存请求正文、文件、图片、音频或模型输出。除非业务确实需要，不应保存不必要的提示词、文件内容或凭据明文。

## 3. 信息用途与共享

信息用于提供登录、请求转发、计量计费、额度控制、风控、故障排查、客户支持和法律合规。为了完成模型请求，必要的请求内容或元数据可能被发送给用户选择的上游模型服务商；上游服务商的处理规则由其自身政策约束。

运营者不得出售个人信息。除非取得授权、法律要求或为完成用户主动发起的上游请求，运营者不应向无关第三方披露信息。

## 4. 保存与安全

运营者应仅在实现上述目的所需期间保存信息，并在设置页或隐私政策中写明日志、订单和审计记录的保留期限。密码、OAuth 凭据、API Key 和支付信息应使用适当的加密、访问控制和脱敏措施保存；用户也应避免在提示词中提交敏感信息。

## 5. 用户权利与联系

在适用法律允许的范围内，用户可以请求查询、更正、删除或导出与其账号相关的信息，也可以撤回不必要的授权。请通过站点展示的客服微信或其他联系渠道提交请求。运营者应在正式上线前把本段中的联系渠道、主体名称和处理时限改为真实信息。

## 6. 政策更新

政策更新后，站点会更新页面中的日期。重大变更可以通过登录提示、站内公告或注册邮箱通知用户。`

const defaultUsagePolicyMarkdown = `## 1. 允许的使用

用户可以通过 Starbridge AI 进行合法的文本、代码、图像、音频或其他模型请求，并遵守本政策、服务条款、上游平台规则及适用法律。

## 2. 禁止的使用

禁止以下行为：

- 违法、欺诈、侵害他人权利或规避监管的活动；
- 试图绕过认证、额度、速率、地区或安全控制；
- 共享、出售、转让或批量测试他人账号、API Key、OAuth 令牌或导入配置；
- 发送恶意扫描、垃圾请求、异常并发、自动化刷量或影响其他用户的流量；
- 上传恶意代码、窃取凭据、侵犯版权、隐私或其他知识产权；
- 利用服务生成需要专业人士审核却故意冒充专业意见的内容，并将其直接用于高风险决策。

## 3. 风险处置

运营者可以根据错误率、异常用量、账号额度、上游反馈和风控规则进行降速、拒绝、切换账号、暂停 Key 或冻结账号。处置范围会尽量与风险相称；用户可以通过客服渠道申请复核。

## 4. 上游规则

不同模型和账号可能有独立的内容政策、区域限制和并发限制。用户使用某个模型时，应同时遵守对应上游服务商的规则；本政策不能扩大用户在上游平台拥有的权限。`

const defaultSupportedRegionsMarkdown = `## 1. 可用地区

Starbridge AI 的可用地区由站点运营者、上游模型服务商、支付渠道和适用法律共同决定。当前站点应在发布前列出实际支持的国家或地区，并说明注册、付款、模型访问和客服是否存在差异。

## 2. 地区限制

部分模型、账号类型、支付方式或 API 可能因出口管制、制裁、服务商政策、网络条件或许可证限制而不可用。用户不得使用代理、虚假资料或其他方式规避适用限制。

## 3. 变更

支持范围可能随上游政策、法律和运营能力变化。页面显示的地区说明仅代表站点当前配置；请求被上游拒绝时，以实际响应为准。`

const defaultServiceSpecificTermsMarkdown = `## 1. API 与客户端连接

用户可以在 API Key 页面获取兼容 OpenAI 的接口地址和使用说明，并将其配置到支持自定义 Endpoint 的客户端。API Key 应按最小权限原则使用，不要粘贴到公开仓库、截图或不受信任的插件中。

## 2. 订阅与账号池

订阅可能绑定余额、模型范围、有效期、分组和平台额度。站点可以在多个已授权上游账号之间进行调度；切换账号不代表上游额度合并，也不保证每次请求由同一账号完成。用户可用额度以账户页面、订阅页面和 API 响应中的最新数据为准。

## 3. 模型与价格

模型列表、模型别名、计费倍率和可用状态由管理员配置。最新模型或高成本模型可能使用单独的套餐、分组或倍率；价格页面展示的金额应以当前站点设置和请求完成时的计费结果为准。

## 4. 维护与异常

上游账号过期、余额不足、风控、限流、网络故障或模型下线都可能导致请求失败。系统会尽可能选择其他可用账号，但不承诺无缝切换或服务连续性。发生争议时，以可核验的请求记录、用量记录和订单记录为依据。`

func normalizeLoginAgreementDocumentID(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	var b strings.Builder
	lastSeparator := false
	for _, r := range raw {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			_, _ = b.WriteRune(r)
			lastSeparator = false
			continue
		}
		if r == '-' || r == '_' || r == ' ' || r == '.' || r == '/' {
			if !lastSeparator && b.Len() > 0 {
				if r == '_' {
					_, _ = b.WriteRune('_')
				} else {
					_, _ = b.WriteRune('-')
				}
				lastSeparator = true
			}
		}
	}
	return strings.Trim(b.String(), "-_")
}

func normalizeLoginAgreementDocuments(docs []LoginAgreementDocument) []LoginAgreementDocument {
	normalized := make([]LoginAgreementDocument, 0, len(docs))
	seen := make(map[string]int, len(docs))
	for i, doc := range docs {
		title := strings.TrimSpace(doc.Title)
		content := strings.TrimSpace(doc.ContentMD)
		if title == "" && content == "" {
			continue
		}
		id := normalizeLoginAgreementDocumentID(doc.ID)
		if id == "" {
			sum := sha256.Sum256([]byte(fmt.Sprintf("%d:%s:%s", i, title, content)))
			id = hex.EncodeToString(sum[:])[:12]
		}
		baseID := id
		for suffix := 2; seen[id] > 0; suffix++ {
			id = fmt.Sprintf("%s-%d", baseID, suffix)
		}
		seen[id]++
		normalized = append(normalized, LoginAgreementDocument{
			ID:        id,
			Title:     title,
			ContentMD: content,
		})
	}
	return normalized
}

func parseLoginAgreementDocuments(raw string) []LoginAgreementDocument {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaultLoginAgreementDocuments()
	}
	var docs []LoginAgreementDocument
	if err := json.Unmarshal([]byte(raw), &docs); err != nil {
		return defaultLoginAgreementDocuments()
	}
	docs = normalizeLoginAgreementDocuments(docs)
	if len(docs) == 0 {
		return defaultLoginAgreementDocuments()
	}
	return docs
}

func marshalLoginAgreementDocuments(docs []LoginAgreementDocument) (string, error) {
	normalized := normalizeLoginAgreementDocuments(docs)
	if len(normalized) == 0 {
		normalized = defaultLoginAgreementDocuments()
	}
	b, err := json.Marshal(normalized)
	if err != nil {
		return "", fmt.Errorf("marshal login agreement documents: %w", err)
	}
	return string(b), nil
}

func buildLoginAgreementRevision(updatedAt string, docs []LoginAgreementDocument) string {
	normalized := normalizeLoginAgreementDocuments(docs)
	payload, err := json.Marshal(struct {
		UpdatedAt string                   `json:"updated_at"`
		Documents []LoginAgreementDocument `json:"documents"`
	}{
		UpdatedAt: strings.TrimSpace(updatedAt),
		Documents: normalized,
	})
	if err != nil {
		payload = []byte(strings.TrimSpace(updatedAt))
	}
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])[:16]
}

// GetFrontendURL 获取前端基础URL（数据库优先，fallback 到配置文件）
func (s *SettingService) GetFrontendURL(ctx context.Context) string {
	val, err := s.settingRepo.GetValue(ctx, SettingKeyFrontendURL)
	if err == nil && strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val)
	}
	return s.cfg.Server.FrontendURL
}

// GetPublicSettings 获取公开设置（无需登录）
func (s *SettingService) GetPublicSettings(ctx context.Context) (*PublicSettings, error) {
	keys := []string{
		SettingKeyRegistrationEnabled,
		SettingKeyEmailVerifyEnabled,
		SettingKeyForceEmailOnThirdPartySignup,
		SettingKeyRegistrationEmailSuffixWhitelist,
		SettingKeyRegistrationEmailDomainQuotaEnabled,
		SettingKeyPromoCodeEnabled,
		SettingKeyPasswordResetEnabled,
		SettingKeyInvitationCodeEnabled,
		SettingKeyTotpEnabled,
		SettingKeyPasskeyEnabled,
		SettingKeyLoginAgreementEnabled,
		SettingKeyLoginAgreementMode,
		SettingKeyLoginAgreementUpdatedAt,
		SettingKeyLoginAgreementDocuments,
		SettingKeyTurnstileEnabled,
		SettingKeyTurnstileSiteKey,
		SettingKeyTencentCaptchaEnabled,
		SettingKeyTencentCaptchaAppID,
		SettingKeyTencentCaptchaRegion,
		SettingKeyAliyunCaptchaEnabled,
		SettingKeyAliyunCaptchaSceneID,
		SettingKeyAliyunCaptchaPrefix,
		SettingKeyAliyunCaptchaRegion,
		SettingKeyAPIKeyACLTrustForwardedIP,
		SettingKeySiteName,
		SettingKeySiteLogo,
		SettingKeySiteSubtitle,
		SettingKeyAPIBaseURL,
		SettingKeyContactInfo,
		SettingKeyDocURL,
		SettingKeyHomeContent,
		SettingKeyCompactHomeEnabled,
		SettingKeyHideCcsImportButton,
		SettingKeyPurchaseSubscriptionEnabled,
		SettingKeyPurchaseSubscriptionURL,
		SettingKeyTableDefaultPageSize,
		SettingKeyTablePageSizeOptions,
		SettingKeyCustomMenuItems,
		SettingKeyCustomEndpoints,
		SettingKeyLinuxDoConnectEnabled,
		SettingKeyDingTalkConnectEnabled,
		SettingKeyWeChatConnectEnabled,
		SettingKeyWeChatConnectAppID,
		SettingKeyWeChatConnectAppSecret,
		SettingKeyWeChatConnectOpenAppID,
		SettingKeyWeChatConnectOpenAppSecret,
		SettingKeyWeChatConnectMPAppID,
		SettingKeyWeChatConnectMPAppSecret,
		SettingKeyWeChatConnectMobileAppID,
		SettingKeyWeChatConnectMobileAppSecret,
		SettingKeyWeChatConnectOpenEnabled,
		SettingKeyWeChatConnectMPEnabled,
		SettingKeyWeChatConnectMobileEnabled,
		SettingKeyWeChatConnectMode,
		SettingKeyWeChatConnectScopes,
		SettingKeyWeChatConnectRedirectURL,
		SettingKeyWeChatConnectFrontendRedirectURL,
		SettingKeyBackendModeEnabled,
		SettingPaymentEnabled,
		SettingBalancePayDisabled,
		SettingKeyOIDCConnectEnabled,
		SettingKeyOIDCConnectProviderName,
		SettingKeyGitHubOAuthEnabled,
		SettingKeyGitHubOAuthClientID,
		SettingKeyGitHubOAuthClientSecret,
		SettingKeyGoogleOAuthEnabled,
		SettingKeyGoogleOAuthClientID,
		SettingKeyGoogleOAuthClientSecret,
		SettingKeyBalanceLowNotifyEnabled,
		SettingKeyBalanceLowNotifyThreshold,
		SettingKeyBalanceLowNotifyRechargeURL,
		SettingKeyAccountQuotaNotifyEnabled,
		SettingKeyChannelMonitorEnabled,
		SettingKeyChannelMonitorMode,
		SettingKeyChannelMonitorDefaultIntervalSeconds,
		SettingKeyChannelMonitorHideThroughput,
		SettingKeyChannelMonitorShowQuota,
		SettingKeyChannelMonitorHideUserRanking,
		SettingKeyAvailableChannelsEnabled,
		SettingKeySubscriptionEnabled,
		SettingKeyModelPlazaEnabled,
		SettingKeyModelPlazaRequireAuth,
		SettingKeyPluginManagementEnabled,
		SettingKeyAffiliateEnabled,
		SettingKeyRiskControlEnabled,
		SettingKeyAllowUserViewErrorRequests,
	}

	settings, err := s.settingRepo.GetMultiple(ctx, keys)
	if err != nil {
		return nil, fmt.Errorf("get public settings: %w", err)
	}

	linuxDoEnabled := false
	if raw, ok := settings[SettingKeyLinuxDoConnectEnabled]; ok {
		linuxDoEnabled = raw == "true"
	} else {
		linuxDoEnabled = s.cfg != nil && s.cfg.LinuxDo.Enabled
	}
	dingTalkEnabled := false
	if raw, ok := settings[SettingKeyDingTalkConnectEnabled]; ok {
		dingTalkEnabled = raw == "true"
	} else {
		dingTalkEnabled = s.cfg != nil && s.cfg.DingTalk.Enabled
	}
	oidcEnabled := false
	if raw, ok := settings[SettingKeyOIDCConnectEnabled]; ok {
		oidcEnabled = raw == "true"
	} else {
		oidcEnabled = s.cfg != nil && s.cfg.OIDC.Enabled
	}
	oidcProviderName := strings.TrimSpace(settings[SettingKeyOIDCConnectProviderName])
	if oidcProviderName == "" && s.cfg != nil {
		oidcProviderName = strings.TrimSpace(s.cfg.OIDC.ProviderName)
	}
	if oidcProviderName == "" {
		oidcProviderName = "OIDC"
	}
	gitHubEnabled := s.emailOAuthPublicEnabled(settings, "github")
	googleEnabled := s.emailOAuthPublicEnabled(settings, "google")
	weChatEnabled, weChatOpenEnabled, weChatMPEnabled, weChatMobileEnabled := s.weChatOAuthCapabilitiesFromSettings(settings)

	// Password reset requires email verification to be enabled
	emailVerifyEnabled := settings[SettingKeyEmailVerifyEnabled] == "true"
	passwordResetEnabled := emailVerifyEnabled && settings[SettingKeyPasswordResetEnabled] == "true"
	registrationEmailSuffixWhitelist := ParseRegistrationEmailSuffixWhitelist(
		settings[SettingKeyRegistrationEmailSuffixWhitelist],
	)
	tableDefaultPageSize, tablePageSizeOptions := parseTablePreferences(
		settings[SettingKeyTableDefaultPageSize],
		settings[SettingKeyTablePageSizeOptions],
	)
	loginAgreementDocuments := parseLoginAgreementDocuments(settings[SettingKeyLoginAgreementDocuments])
	loginAgreementUpdatedAt := strings.TrimSpace(settings[SettingKeyLoginAgreementUpdatedAt])
	if loginAgreementUpdatedAt == "" {
		loginAgreementUpdatedAt = defaultLoginAgreementDate
	}

	var balanceLowNotifyThreshold float64
	if v, err := strconv.ParseFloat(settings[SettingKeyBalanceLowNotifyThreshold], 64); err == nil && v >= 0 {
		balanceLowNotifyThreshold = v
	}

	return &PublicSettings{
		RegistrationEnabled:                 settings[SettingKeyRegistrationEnabled] == "true",
		EmailVerifyEnabled:                  emailVerifyEnabled,
		ForceEmailOnThirdPartySignup:        settings[SettingKeyForceEmailOnThirdPartySignup] == "true",
		RegistrationEmailSuffixWhitelist:    registrationEmailSuffixWhitelist,
		RegistrationEmailDomainQuotaEnabled: settings[SettingKeyRegistrationEmailDomainQuotaEnabled] == "true",
		PromoCodeEnabled:                    settings[SettingKeyPromoCodeEnabled] != "false", // 默认启用
		PasswordResetEnabled:                passwordResetEnabled,
		InvitationCodeEnabled:               settings[SettingKeyInvitationCodeEnabled] == "true",
		TotpEnabled:                         settings[SettingKeyTotpEnabled] == "true",
		PasskeyEnabled:                      s.passkeyConfigured() && s.passkeySettingEnabled(settings),
		LoginAgreementEnabled:               settings[SettingKeyLoginAgreementEnabled] == "true" && len(loginAgreementDocuments) > 0,
		LoginAgreementMode:                  normalizeLoginAgreementMode(settings[SettingKeyLoginAgreementMode]),
		LoginAgreementUpdatedAt:             loginAgreementUpdatedAt,
		LoginAgreementRevision:              buildLoginAgreementRevision(loginAgreementUpdatedAt, loginAgreementDocuments),
		LoginAgreementDocuments:             loginAgreementDocuments,
		TurnstileEnabled:                    settings[SettingKeyTurnstileEnabled] == "true",
		TurnstileSiteKey:                    settings[SettingKeyTurnstileSiteKey],
		TencentCaptchaEnabled:               settings[SettingKeyTencentCaptchaEnabled] == "true",
		TencentCaptchaAppID:                 settings[SettingKeyTencentCaptchaAppID],
		TencentCaptchaRegion:                normalizeTencentCaptchaRegion(settings[SettingKeyTencentCaptchaRegion]),
		AliyunCaptchaEnabled:                settings[SettingKeyAliyunCaptchaEnabled] == "true",
		AliyunCaptchaSceneID:                settings[SettingKeyAliyunCaptchaSceneID],
		AliyunCaptchaPrefix:                 settings[SettingKeyAliyunCaptchaPrefix],
		AliyunCaptchaRegion:                 normalizeAliyunCaptchaRegion(settings[SettingKeyAliyunCaptchaRegion]),
		SiteName:                            s.getStringOrDefault(settings, SettingKeySiteName, "Starbridge AI"),
		SiteLogo:                            settings[SettingKeySiteLogo],
		SiteSubtitle:                        s.getStringOrDefault(settings, SettingKeySiteSubtitle, "Multi-model AI API Gateway"),
		APIBaseURL:                          settings[SettingKeyAPIBaseURL],
		ContactInfo:                         settings[SettingKeyContactInfo],
		DocURL:                              settings[SettingKeyDocURL],
		HomeContent:                         settings[SettingKeyHomeContent],
		CompactHomeEnabled:                  settings[SettingKeyCompactHomeEnabled] == "true",
		HideCcsImportButton:                 settings[SettingKeyHideCcsImportButton] == "true",
		PurchaseSubscriptionEnabled:         settings[SettingKeyPurchaseSubscriptionEnabled] == "true",
		PurchaseSubscriptionURL:             strings.TrimSpace(settings[SettingKeyPurchaseSubscriptionURL]),
		TableDefaultPageSize:                tableDefaultPageSize,
		TablePageSizeOptions:                tablePageSizeOptions,
		CustomMenuItems:                     settings[SettingKeyCustomMenuItems],
		CustomEndpoints:                     settings[SettingKeyCustomEndpoints],
		LinuxDoOAuthEnabled:                 linuxDoEnabled,
		DingTalkOAuthEnabled:                dingTalkEnabled,
		WeChatOAuthEnabled:                  weChatEnabled,
		WeChatOAuthOpenEnabled:              weChatOpenEnabled,
		WeChatOAuthMPEnabled:                weChatMPEnabled,
		WeChatOAuthMobileEnabled:            weChatMobileEnabled,
		BackendModeEnabled:                  settings[SettingKeyBackendModeEnabled] == "true",
		PaymentEnabled:                      settings[SettingPaymentEnabled] == "true",
		// Keep the public billing mode aligned with payment-config parsing: only an
		// explicit true disables balance recharge.
		PaymentBalanceDisabled:              settings[SettingBalancePayDisabled] == "true",
		OIDCOAuthEnabled:                    oidcEnabled,
		OIDCOAuthProviderName:               oidcProviderName,
		GitHubOAuthEnabled:                  gitHubEnabled,
		GoogleOAuthEnabled:                  googleEnabled,
		BalanceLowNotifyEnabled:             settings[SettingKeyBalanceLowNotifyEnabled] == "true",
		AccountQuotaNotifyEnabled:           settings[SettingKeyAccountQuotaNotifyEnabled] == "true",
		BalanceLowNotifyThreshold:           balanceLowNotifyThreshold,
		BalanceLowNotifyRechargeURL:         settings[SettingKeyBalanceLowNotifyRechargeURL],

		ChannelMonitorEnabled:                !isFalseSettingValue(settings[SettingKeyChannelMonitorEnabled]),
		ChannelMonitorMode:                   normalizeChannelMonitorMode(settings[SettingKeyChannelMonitorMode]),
		ChannelMonitorDefaultIntervalSeconds: parseChannelMonitorInterval(settings[SettingKeyChannelMonitorDefaultIntervalSeconds]),
		ChannelMonitorHideThroughput:         !isFalseSettingValue(settings[SettingKeyChannelMonitorHideThroughput]),
		ChannelMonitorShowQuota:              settings[SettingKeyChannelMonitorShowQuota] == "true",
		ChannelMonitorHideUserRanking:        isTrueSettingValue(settings[SettingKeyChannelMonitorHideUserRanking]),

		AvailableChannelsEnabled: settings[SettingKeyAvailableChannelsEnabled] == "true",

		SubscriptionEnabled: !isFalseSettingValue(settings[SettingKeySubscriptionEnabled]),

		ModelPlazaEnabled:       settings[SettingKeyModelPlazaEnabled] == "true",
		ModelPlazaRequireAuth:   settings[SettingKeyModelPlazaRequireAuth] == "true",
		PluginManagementEnabled: settings[SettingKeyPluginManagementEnabled] == "true",

		AffiliateEnabled: settings[SettingKeyAffiliateEnabled] == "true",

		RiskControlEnabled: settings[SettingKeyRiskControlEnabled] == "true",

		AllowUserViewErrorRequests: settings[SettingKeyAllowUserViewErrorRequests] == "true",
	}, nil
}

// channelMonitorIntervalMin / channelMonitorIntervalMax bound the default interval
// (mirrors the monitor-level constraint but lives here so setting_service stays decoupled).
const (
	channelMonitorIntervalMin      = 15
	channelMonitorIntervalMax      = 3600
	channelMonitorIntervalFallback = 60
	defaultChannelMonitorMode      = ChannelMonitorModeV1
)

// normalizeChannelMonitorMode accepts only v1/v2; empty/invalid → v1 (safe default).
func normalizeChannelMonitorMode(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case ChannelMonitorModeV1, "":
		return ChannelMonitorModeV1
	case ChannelMonitorModeV2:
		return ChannelMonitorModeV2
	default:
		return defaultChannelMonitorMode
	}
}

// parseChannelMonitorInterval parses the stored string and clamps to [15, 3600].
// Empty / invalid input falls back to channelMonitorIntervalFallback.
func parseChannelMonitorInterval(raw string) int {
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		return channelMonitorIntervalFallback
	}
	return clampChannelMonitorInterval(v)
}

// clampChannelMonitorInterval clamps v to the allowed range. 0 means "not provided".
func clampChannelMonitorInterval(v int) int {
	if v <= 0 {
		return 0
	}
	if v < channelMonitorIntervalMin {
		return channelMonitorIntervalMin
	}
	if v > channelMonitorIntervalMax {
		return channelMonitorIntervalMax
	}
	return v
}

// ChannelMonitorRuntime is the lightweight view of the channel monitor feature
// consumed by the runner, V2 aggregator, and user-facing handlers.
type ChannelMonitorRuntime struct {
	Enabled                bool
	Mode                   string // ChannelMonitorModeV1 or ChannelMonitorModeV2
	DefaultIntervalSeconds int
	// HideThroughput: when true, user-facing V2 APIs omit RPM/TPM scale signals.
	HideThroughput bool
	// ShowQuota: when true, user-facing monitor views keep the quota/balance
	// snapshots; otherwise the user handler strips them server-side.
	// Parsed fail-closed (only literal "true" enables). Admin always sees them.
	ShowQuota bool
	// HideUserRanking: when true, user-facing V2 views hide the user ranking tab
	// and the /users payload. Parsed fail-open (only literal "true" hides it).
	HideUserRanking bool
}

// ActiveProbesAllowed reports whether V1 active provider probes may run.
func (r ChannelMonitorRuntime) ActiveProbesAllowed() bool {
	return r.Enabled && r.Mode == ChannelMonitorModeV1
}

// PassiveAggregationAllowed reports whether V2 passive aggregation may run.
func (r ChannelMonitorRuntime) PassiveAggregationAllowed() bool {
	return r.Enabled && r.Mode == ChannelMonitorModeV2
}

// GetChannelMonitorRuntime reads the channel monitor feature flags directly from
// the settings store. Fail-open: on error returns Enabled=true, Mode=v1, default interval.
func (s *SettingService) GetChannelMonitorRuntime(ctx context.Context) ChannelMonitorRuntime {
	if s == nil || s.settingRepo == nil {
		return ChannelMonitorRuntime{
			Enabled:                true,
			Mode:                   defaultChannelMonitorMode,
			DefaultIntervalSeconds: channelMonitorIntervalFallback,
			HideThroughput:         true,
		}
	}
	vals, err := s.settingRepo.GetMultiple(ctx, []string{
		SettingKeyChannelMonitorEnabled,
		SettingKeyChannelMonitorMode,
		SettingKeyChannelMonitorDefaultIntervalSeconds,
		SettingKeyChannelMonitorHideThroughput,
		SettingKeyChannelMonitorShowQuota,
		SettingKeyChannelMonitorHideUserRanking,
	})
	if err != nil {
		return ChannelMonitorRuntime{
			Enabled:                true,
			Mode:                   defaultChannelMonitorMode,
			DefaultIntervalSeconds: channelMonitorIntervalFallback,
			HideThroughput:         true,
		}
	}
	return ChannelMonitorRuntime{
		Enabled:                !isFalseSettingValue(vals[SettingKeyChannelMonitorEnabled]),
		Mode:                   normalizeChannelMonitorMode(vals[SettingKeyChannelMonitorMode]),
		DefaultIntervalSeconds: parseChannelMonitorInterval(vals[SettingKeyChannelMonitorDefaultIntervalSeconds]),
		HideThroughput:         !isFalseSettingValue(vals[SettingKeyChannelMonitorHideThroughput]),
		ShowQuota:              vals[SettingKeyChannelMonitorShowQuota] == "true",
		HideUserRanking:        isTrueSettingValue(vals[SettingKeyChannelMonitorHideUserRanking]),
	}
}

// AvailableChannelsRuntime is the lightweight view of the available-channels feature
// switch consumed by the user-facing handler.
type AvailableChannelsRuntime struct {
	Enabled bool
}

// GetAvailableChannelsRuntime reads the available-channels feature switch directly
// from the settings store. Fail-closed: on error returns Enabled=false, matching
// the opt-in default (unknown ↔ disabled).
func (s *SettingService) GetAvailableChannelsRuntime(ctx context.Context) AvailableChannelsRuntime {
	vals, err := s.settingRepo.GetMultiple(ctx, []string{SettingKeyAvailableChannelsEnabled})
	if err != nil {
		return AvailableChannelsRuntime{Enabled: false}
	}
	return AvailableChannelsRuntime{
		Enabled: vals[SettingKeyAvailableChannelsEnabled] == "true",
	}
}

// ModelPlazaRuntime is the lightweight view of the model-plaza feature consumed
// by the public plaza handler.
type ModelPlazaRuntime struct {
	Enabled     bool
	RequireAuth bool
	Description string
}

// GetModelPlazaRuntime reads the model-plaza feature switches directly from the
// settings store. Fail-closed: on error returns Enabled=false, matching the
// opt-in default (unknown ↔ disabled).
func (s *SettingService) GetModelPlazaRuntime(ctx context.Context) ModelPlazaRuntime {
	vals, err := s.settingRepo.GetMultiple(ctx, []string{
		SettingKeyModelPlazaEnabled,
		SettingKeyModelPlazaRequireAuth,
		SettingKeyModelPlazaDescription,
	})
	if err != nil {
		return ModelPlazaRuntime{Enabled: false}
	}
	return ModelPlazaRuntime{
		Enabled:     vals[SettingKeyModelPlazaEnabled] == "true",
		RequireAuth: vals[SettingKeyModelPlazaRequireAuth] == "true",
		Description: vals[SettingKeyModelPlazaDescription],
	}
}

// IsUserErrorViewAllowed reads the user-facing error-requests visibility switch
// directly from the settings store. Fail-closed: on error returns false (opt-in default).
func (s *SettingService) IsUserErrorViewAllowed(ctx context.Context) bool {
	vals, err := s.settingRepo.GetMultiple(ctx, []string{SettingKeyAllowUserViewErrorRequests})
	if err != nil {
		slog.Warn("failed to get allow_user_view_error_requests setting, defaulting to false", "error", err)
		return false
	}
	return vals[SettingKeyAllowUserViewErrorRequests] == "true"
}

// PublicSettingsInjectionPayload is the JSON shape embedded into HTML as
// `window.__APP_CONFIG__` so the frontend can hydrate feature flags & site
// config before the first XHR finishes.
//
// INVARIANT: every `json` tag here MUST also exist on handler/dto.PublicSettings.
// If you forget a feature-flag field here, the frontend's
// `cachedPublicSettings.xxx_enabled` will be `undefined` on refresh until the
// async `/api/v1/settings/public` call returns — which causes opt-in menus
// (strict `=== true`) to flicker off/on. See
// frontend/src/utils/featureFlags.ts for the matching registry.
//
// A unit test diffs this struct's JSON keys against dto.PublicSettings to catch
// drift automatically (see setting_service_injection_test.go).
type PublicSettingsInjectionPayload struct {
	RegistrationEnabled                 bool                     `json:"registration_enabled"`
	EmailVerifyEnabled                  bool                     `json:"email_verify_enabled"`
	RegistrationEmailSuffixWhitelist    []string                 `json:"registration_email_suffix_whitelist"`
	RegistrationEmailDomainQuotaEnabled bool                     `json:"registration_email_domain_quota_enabled"`
	PromoCodeEnabled                    bool                     `json:"promo_code_enabled"`
	PasswordResetEnabled                bool                     `json:"password_reset_enabled"`
	InvitationCodeEnabled               bool                     `json:"invitation_code_enabled"`
	TotpEnabled                         bool                     `json:"totp_enabled"`
	PasskeyEnabled                      bool                     `json:"passkey_enabled"`
	LoginAgreementEnabled               bool                     `json:"login_agreement_enabled"`
	LoginAgreementMode                  string                   `json:"login_agreement_mode"`
	LoginAgreementUpdatedAt             string                   `json:"login_agreement_updated_at"`
	LoginAgreementRevision              string                   `json:"login_agreement_revision"`
	LoginAgreementDocuments             []LoginAgreementDocument `json:"login_agreement_documents"`
	TurnstileEnabled                    bool                     `json:"turnstile_enabled"`
	TurnstileSiteKey                    string                   `json:"turnstile_site_key"`
	TencentCaptchaEnabled               bool                     `json:"tencent_captcha_enabled"`
	TencentCaptchaAppID                 string                   `json:"tencent_captcha_app_id"`
	TencentCaptchaRegion                string                   `json:"tencent_captcha_region"`
	AliyunCaptchaEnabled                bool                     `json:"aliyun_captcha_enabled"`
	AliyunCaptchaSceneID                string                   `json:"aliyun_captcha_scene_id"`
	AliyunCaptchaPrefix                 string                   `json:"aliyun_captcha_prefix"`
	AliyunCaptchaRegion                 string                   `json:"aliyun_captcha_region"`
	SiteName                            string                   `json:"site_name"`
	SiteLogo                            string                   `json:"site_logo"`
	SiteSubtitle                        string                   `json:"site_subtitle"`
	APIBaseURL                          string                   `json:"api_base_url"`
	ContactInfo                         string                   `json:"contact_info"`
	DocURL                              string                   `json:"doc_url"`
	HomeContent                         string                   `json:"home_content"`
	CompactHomeEnabled                  bool                     `json:"compact_home_enabled"`
	HideCcsImportButton                 bool                     `json:"hide_ccs_import_button"`
	PurchaseSubscriptionEnabled         bool                     `json:"purchase_subscription_enabled"`
	PurchaseSubscriptionURL             string                   `json:"purchase_subscription_url"`
	TableDefaultPageSize                int                      `json:"table_default_page_size"`
	TablePageSizeOptions                []int                    `json:"table_page_size_options"`
	CustomMenuItems                     json.RawMessage          `json:"custom_menu_items"`
	CustomEndpoints                     json.RawMessage          `json:"custom_endpoints"`
	LinuxDoOAuthEnabled                 bool                     `json:"linuxdo_oauth_enabled"`
	DingTalkOAuthEnabled                bool                     `json:"dingtalk_oauth_enabled"`
	WeChatOAuthEnabled                  bool                     `json:"wechat_oauth_enabled"`
	WeChatOAuthOpenEnabled              bool                     `json:"wechat_oauth_open_enabled"`
	WeChatOAuthMPEnabled                bool                     `json:"wechat_oauth_mp_enabled"`
	WeChatOAuthMobileEnabled            bool                     `json:"wechat_oauth_mobile_enabled"`
	OIDCOAuthEnabled                    bool                     `json:"oidc_oauth_enabled"`
	OIDCOAuthProviderName               string                   `json:"oidc_oauth_provider_name"`
	GitHubOAuthEnabled                  bool                     `json:"github_oauth_enabled"`
	GoogleOAuthEnabled                  bool                     `json:"google_oauth_enabled"`
	BackendModeEnabled                  bool                     `json:"backend_mode_enabled"`
	PaymentEnabled                      bool                     `json:"payment_enabled"`
	PaymentBalanceDisabled              bool                     `json:"payment_balance_disabled"`
	Version                             string                   `json:"version"`
	// 服务器全局时区（IANA 名称与当前 UTC 偏移），高峰时段等服务端本地时间窗口的展示标注用
	ServerTimezone              string  `json:"server_timezone"`
	ServerUTCOffset             string  `json:"server_utc_offset"`
	BalanceLowNotifyEnabled     bool    `json:"balance_low_notify_enabled"`
	AccountQuotaNotifyEnabled   bool    `json:"account_quota_notify_enabled"`
	BalanceLowNotifyThreshold   float64 `json:"balance_low_notify_threshold"`
	BalanceLowNotifyRechargeURL string  `json:"balance_low_notify_recharge_url"`

	// Feature flags — MUST match the opt-in/opt-out registry in
	// frontend/src/utils/featureFlags.ts. Missing a field here is the bug
	// that hid the "可用渠道" menu on page refresh.
	ChannelMonitorEnabled                bool   `json:"channel_monitor_enabled"`
	ChannelMonitorMode                   string `json:"channel_monitor_mode"`
	ChannelMonitorDefaultIntervalSeconds int    `json:"channel_monitor_default_interval_seconds"`
	// ChannelMonitorHideThroughput is public so the user UI can hide RPM/TPM
	// without waiting for API redaction alone (defense in depth).
	ChannelMonitorHideThroughput bool `json:"channel_monitor_hide_throughput"`
	// ChannelMonitorShowQuota gates the user-facing quota/balance display on
	// monitors; fail-closed (absent/false = hidden). Admin UI always shows it.
	// ChannelMonitorHideUserRanking hides the user ranking tab and /users payload
	// from non-admin channel-monitor v2 viewers; default false (visible).
	ChannelMonitorHideUserRanking bool `json:"channel_monitor_hide_user_ranking"`
	ChannelMonitorShowQuota       bool `json:"channel_monitor_show_quota"`
	AvailableChannelsEnabled      bool `json:"available_channels_enabled"`
	SubscriptionEnabled           bool `json:"subscription_enabled"`
	ModelPlazaEnabled             bool `json:"model_plaza_enabled"`
	ModelPlazaRequireAuth         bool `json:"model_plaza_require_auth"`
	PluginManagementEnabled       bool `json:"plugin_management_enabled"`
	AffiliateEnabled              bool `json:"affiliate_enabled"`
	RiskControlEnabled            bool `json:"risk_control_enabled"`
	AllowUserViewErrorRequests    bool `json:"allow_user_view_error_requests"`
}

// GetPublicSettingsForInjection returns public settings in a format suitable for HTML injection.
// This implements the web.PublicSettingsProvider interface.
func (s *SettingService) GetPublicSettingsForInjection(ctx context.Context) (any, error) {
	settings, err := s.GetPublicSettings(ctx)
	if err != nil {
		return nil, err
	}

	return &PublicSettingsInjectionPayload{
		RegistrationEnabled:                 settings.RegistrationEnabled,
		EmailVerifyEnabled:                  settings.EmailVerifyEnabled,
		RegistrationEmailSuffixWhitelist:    settings.RegistrationEmailSuffixWhitelist,
		RegistrationEmailDomainQuotaEnabled: settings.RegistrationEmailDomainQuotaEnabled,
		PromoCodeEnabled:                    settings.PromoCodeEnabled,
		PasswordResetEnabled:                settings.PasswordResetEnabled,
		InvitationCodeEnabled:               settings.InvitationCodeEnabled,
		TotpEnabled:                         settings.TotpEnabled,
		PasskeyEnabled:                      settings.PasskeyEnabled,
		LoginAgreementEnabled:               settings.LoginAgreementEnabled,
		LoginAgreementMode:                  settings.LoginAgreementMode,
		LoginAgreementUpdatedAt:             settings.LoginAgreementUpdatedAt,
		LoginAgreementRevision:              settings.LoginAgreementRevision,
		LoginAgreementDocuments:             settings.LoginAgreementDocuments,
		TurnstileEnabled:                    settings.TurnstileEnabled,
		TurnstileSiteKey:                    settings.TurnstileSiteKey,
		TencentCaptchaEnabled:               settings.TencentCaptchaEnabled,
		TencentCaptchaAppID:                 settings.TencentCaptchaAppID,
		TencentCaptchaRegion:                settings.TencentCaptchaRegion,
		AliyunCaptchaEnabled:                settings.AliyunCaptchaEnabled,
		AliyunCaptchaSceneID:                settings.AliyunCaptchaSceneID,
		AliyunCaptchaPrefix:                 settings.AliyunCaptchaPrefix,
		AliyunCaptchaRegion:                 settings.AliyunCaptchaRegion,
		SiteName:                            settings.SiteName,
		SiteLogo:                            settings.SiteLogo,
		SiteSubtitle:                        settings.SiteSubtitle,
		APIBaseURL:                          settings.APIBaseURL,
		ContactInfo:                         settings.ContactInfo,
		DocURL:                              settings.DocURL,
		HomeContent:                         settings.HomeContent,
		CompactHomeEnabled:                  settings.CompactHomeEnabled,
		HideCcsImportButton:                 settings.HideCcsImportButton,
		PurchaseSubscriptionEnabled:         settings.PurchaseSubscriptionEnabled,
		PurchaseSubscriptionURL:             settings.PurchaseSubscriptionURL,
		TableDefaultPageSize:                settings.TableDefaultPageSize,
		TablePageSizeOptions:                settings.TablePageSizeOptions,
		CustomMenuItems:                     filterUserVisibleMenuItems(settings.CustomMenuItems),
		CustomEndpoints:                     safeRawJSONArray(settings.CustomEndpoints),
		LinuxDoOAuthEnabled:                 settings.LinuxDoOAuthEnabled,
		DingTalkOAuthEnabled:                settings.DingTalkOAuthEnabled,
		WeChatOAuthEnabled:                  settings.WeChatOAuthEnabled,
		WeChatOAuthOpenEnabled:              settings.WeChatOAuthOpenEnabled,
		WeChatOAuthMPEnabled:                settings.WeChatOAuthMPEnabled,
		WeChatOAuthMobileEnabled:            settings.WeChatOAuthMobileEnabled,
		OIDCOAuthEnabled:                    settings.OIDCOAuthEnabled,
		OIDCOAuthProviderName:               settings.OIDCOAuthProviderName,
		GitHubOAuthEnabled:                  settings.GitHubOAuthEnabled,
		GoogleOAuthEnabled:                  settings.GoogleOAuthEnabled,
		BackendModeEnabled:                  settings.BackendModeEnabled,
		PaymentEnabled:                      settings.PaymentEnabled,
		PaymentBalanceDisabled:              settings.PaymentBalanceDisabled,
		Version:                             s.version,
		ServerTimezone:                      timezone.Name(),
		ServerUTCOffset:                     timezone.UTCOffset(),
		BalanceLowNotifyEnabled:             settings.BalanceLowNotifyEnabled,
		AccountQuotaNotifyEnabled:           settings.AccountQuotaNotifyEnabled,
		BalanceLowNotifyThreshold:           settings.BalanceLowNotifyThreshold,
		BalanceLowNotifyRechargeURL:         settings.BalanceLowNotifyRechargeURL,

		ChannelMonitorEnabled:                settings.ChannelMonitorEnabled,
		ChannelMonitorMode:                   settings.ChannelMonitorMode,
		ChannelMonitorDefaultIntervalSeconds: settings.ChannelMonitorDefaultIntervalSeconds,
		ChannelMonitorHideThroughput:         settings.ChannelMonitorHideThroughput,
		ChannelMonitorShowQuota:              settings.ChannelMonitorShowQuota,
		ChannelMonitorHideUserRanking:        settings.ChannelMonitorHideUserRanking,
		AvailableChannelsEnabled:             settings.AvailableChannelsEnabled,
		SubscriptionEnabled:                  settings.SubscriptionEnabled,
		ModelPlazaEnabled:                    settings.ModelPlazaEnabled,
		ModelPlazaRequireAuth:                settings.ModelPlazaRequireAuth,
		PluginManagementEnabled:              settings.PluginManagementEnabled,
		AffiliateEnabled:                     settings.AffiliateEnabled,
		RiskControlEnabled:                   settings.RiskControlEnabled,
		AllowUserViewErrorRequests:           settings.AllowUserViewErrorRequests,
	}, nil
}

// filterUserVisibleMenuItems filters out admin-only menu items from a raw JSON
// array string, returning only items with visibility != "admin".
func filterUserVisibleMenuItems(raw string) json.RawMessage {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return json.RawMessage("[]")
	}
	var items []struct {
		Visibility string `json:"visibility"`
	}
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return json.RawMessage("[]")
	}

	// Parse full items to preserve all fields
	var fullItems []json.RawMessage
	if err := json.Unmarshal([]byte(raw), &fullItems); err != nil {
		return json.RawMessage("[]")
	}

	var filtered []json.RawMessage
	for i, item := range items {
		if item.Visibility != "admin" {
			filtered = append(filtered, fullItems[i])
		}
	}
	if len(filtered) == 0 {
		return json.RawMessage("[]")
	}
	result, err := json.Marshal(filtered)
	if err != nil {
		return json.RawMessage("[]")
	}
	return result
}

// safeRawJSONArray returns raw as json.RawMessage if it's valid JSON, otherwise "[]".
func safeRawJSONArray(raw string) json.RawMessage {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return json.RawMessage("[]")
	}
	if json.Valid([]byte(raw)) {
		return json.RawMessage(raw)
	}
	return json.RawMessage("[]")
}

// GetFrameSrcOrigins returns deduplicated http(s) origins from home_content URL,
// purchase_subscription_url, and all custom_menu_items URLs. Used by the router layer for CSP frame-src injection.
func (s *SettingService) GetFrameSrcOrigins(ctx context.Context) ([]string, error) {
	settings, err := s.GetPublicSettings(ctx)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	var origins []string

	addOrigin := func(rawURL string) {
		if origin := extractOriginFromURL(rawURL); origin != "" {
			if _, ok := seen[origin]; !ok {
				seen[origin] = struct{}{}
				origins = append(origins, origin)
			}
		}
	}

	// home content URL (when home_content is set to a URL for iframe embedding)
	addOrigin(settings.HomeContent)

	// purchase subscription URL
	if settings.PurchaseSubscriptionEnabled {
		addOrigin(settings.PurchaseSubscriptionURL)
	}

	// all custom menu items (including admin-only, since CSP must allow all iframes)
	for _, item := range parseCustomMenuItemURLs(settings.CustomMenuItems) {
		addOrigin(item)
	}

	return origins, nil
}

// extractOriginFromURL returns the scheme+host origin from rawURL.
// Only http and https schemes are accepted.
func extractOriginFromURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	u, err := url.Parse(rawURL)
	if err != nil || u.Host == "" {
		return ""
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

// parseCustomMenuItemURLs extracts URLs from a raw JSON array of custom menu items.
func parseCustomMenuItemURLs(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}
	var items []struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	urls := make([]string, 0, len(items))
	for _, item := range items {
		if item.URL != "" {
			urls = append(urls, item.URL)
		}
	}
	return urls
}

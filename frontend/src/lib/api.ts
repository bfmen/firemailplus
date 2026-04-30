/**
 * API 配置和基础请求函数
 */

import type {
  EmailAccount,
  Email,
  EmailAddress,
  EmailStats,
  Folder,
  EmailGroup,
} from '@/types/email';
import {
  getArchiveEmailUrl,
  getBatchDeleteEmailAccountsUrl,
  getBatchEmailOperationsUrl,
  getBatchMarkAccountsAsReadUrl,
  getBatchSyncEmailAccountsUrl,
  getCreateCustomEmailAccountUrl,
  getCreateEmailAccountUrl,
  getCreateEmailGroupUrl,
  getCreateFolderUrl,
  getCreateManualOAuthAccountUrl,
  getCreateOAuthAccountUrl,
  getDeleteEmailAccountUrl,
  getDeleteEmailGroupUrl,
  getDeleteEmailUrl,
  getDeleteDraftUrl,
  getDeleteFolderUrl,
  getDeleteTemplateUrl,
  getDownloadAttachmentUrl,
  getForwardEmailUrl,
  getGetCurrentUserUrl,
  getGetDraftUrl,
  getGetEmailAccountUrl,
  getGetEmailUrl,
  getGetFolderUrl,
  getGetSendStatusUrl,
  getGetTemplateUrl,
  getInitGmailOAuthUrl,
  getInitOutlookOAuthUrl,
  getListEmailAccountsUrl,
  getListDraftsUrl,
  getListEmailGroupsUrl,
  getListEmailsUrl,
  getListFoldersUrl,
  getListTemplatesUrl,
  getLoginUrl,
  getLogoutUrl,
  getMarkAccountAsReadUrl,
  getMarkEmailAsReadUrl,
  getMarkEmailAsUnreadUrl,
  getMarkFolderAsReadUrl,
  getMoveEmailUrl,
  getReorderEmailGroupsUrl,
  getReplyAllEmailUrl,
  getReplyEmailUrl,
  getResendEmailUrl,
  getSaveDraftUrl,
  getSearchEmailsUrl,
  getSendBulkEmailsUrl,
  getSendEmailUrl,
  getSetDefaultEmailGroupUrl,
  getSyncEmailAccountUrl,
  getSyncFolderUrl,
  getTestEmailAccountUrl,
  getToggleEmailStarUrl,
  getUpdateEmailAccountUrl,
  getUpdateEmailGroupUrl,
  getUpdateEmailUrl,
  getUpdateDraftUrl,
  getUpdateFolderUrl,
  getUpdateTemplateUrl,
  getCreateTemplateUrl,
  getHandleOAuthCallbackUrl,
} from '@/api/generated/firemail';

export const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL || 'http://localhost:8080/api/v1';

// API 响应类型
export interface ApiResponse<T = unknown> {
  success: boolean;
  data?: T;
  message?: string;
  error?: string;
}

type ApiError = Error & {
  status?: number;
  data?: unknown;
};

// 认证相关类型
export interface LoginRequest {
  username: string;
  password: string;
}

export interface User {
  id: number;
  username: string;
  email?: string;
  display_name?: string;
  role: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
  last_login_at?: string;
  login_count: number;
}

export interface LoginResponse {
  token: string;
  expires_at: string;
  user: User;
}

// 邮箱账户创建请求类型

export interface CreateAccountRequest {
  name: string;
  email: string;
  provider: string;
  auth_method: string;
  username?: string;
  password?: string;
  imap_host?: string;
  imap_port?: number;
  imap_security?: string;
  smtp_host?: string;
  smtp_port?: number;
  smtp_security?: string;
  group_id?: number;
}

// 基础请求函数
class ApiClient {
  private generatedEndpoint(path: string): string {
    return path.startsWith('/api/v1') ? path.slice('/api/v1'.length) || '/' : path;
  }

  private getAuthToken(): string | null {
    if (typeof window !== 'undefined') {
      // 从Zustand persist存储中获取token
      const authStorage = localStorage.getItem('auth-storage');
      if (authStorage) {
        try {
          const parsed = JSON.parse(authStorage);
          return parsed.state?.token || null;
        } catch (error) {
          console.error('Failed to parse auth storage:', error);
          return null;
        }
      }
    }
    return null;
  }

  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<ApiResponse<T>> {
    const token = this.getAuthToken();
    const url = `${API_BASE_URL}${endpoint}`;

    const config: RequestInit = {
      headers: {
        'Content-Type': 'application/json',
        ...(token && { Authorization: `Bearer ${token}` }),
        ...options.headers,
      },
      ...options,
    };

    try {
      const response = await fetch(url, config);

      // 尝试解析JSON响应
      let data: ApiResponse<T>;
      try {
        data = (await response.json()) as ApiResponse<T>;
      } catch {
        throw new Error(`服务器响应格式错误: ${response.status}`);
      }

      if (!response.ok) {
        // 根据状态码提供更友好的错误消息
        let errorMessage = data.message || data.error || '请求失败';

        switch (response.status) {
          case 401:
            errorMessage = data.message || '登录已过期，请重新登录';
            // 401 错误表示认证失败，需要清除认证状态
            if (typeof window !== 'undefined') {
              // 清除认证存储
              localStorage.removeItem('auth-storage');
              // 延迟重定向，让错误处理完成
              setTimeout(() => {
                window.location.href = '/login';
              }, 100);
            }
            break;
          case 403:
            errorMessage = data.message || '账户已被禁用';
            break;
          case 404:
            errorMessage = data.message || '请求的资源不存在';
            break;
          case 500:
            errorMessage = data.message || '服务器内部错误';
            break;
          default:
            errorMessage = data.message || `请求失败 (${response.status})`;
        }

        const apiError = new Error(errorMessage) as ApiError;
        apiError.status = response.status;
        apiError.data = data;
        throw apiError;
      }

      return data;
    } catch (error) {
      console.error('API request failed:', error);
      throw error;
    }
  }

  // 认证相关 API
  async login(credentials: LoginRequest): Promise<ApiResponse<LoginResponse>> {
    return this.request<LoginResponse>(this.generatedEndpoint(getLoginUrl()), {
      method: 'POST',
      body: JSON.stringify(credentials),
    });
  }

  async logout(): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getLogoutUrl()), {
      method: 'POST',
    });
  }

  async getCurrentUser(): Promise<ApiResponse<LoginResponse['user']>> {
    return this.request(this.generatedEndpoint(getGetCurrentUserUrl()));
  }

  // 邮箱账户相关 API
  async getEmailAccounts(): Promise<ApiResponse<EmailAccount[]>> {
    return this.request(this.generatedEndpoint(getListEmailAccountsUrl()));
  }

  async getEmailAccount(id: number): Promise<ApiResponse<EmailAccount>> {
    return this.request(this.generatedEndpoint(getGetEmailAccountUrl(id)));
  }

  async createEmailAccount(account: CreateAccountRequest): Promise<ApiResponse<EmailAccount>> {
    return this.request<EmailAccount>(this.generatedEndpoint(getCreateEmailAccountUrl()), {
      method: 'POST',
      body: JSON.stringify(account),
    });
  }

  async updateEmailAccount(
    id: number,
    data: {
      name?: string;
      password?: string;
      imap_host?: string;
      imap_port?: number;
      imap_security?: string;
      smtp_host?: string;
      smtp_port?: number;
      smtp_security?: string;
      is_active?: boolean;
      group_id?: number | null;
    }
  ): Promise<ApiResponse<EmailAccount>> {
    return this.request<EmailAccount>(this.generatedEndpoint(getUpdateEmailAccountUrl(id)), {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async testEmailAccount(id: number): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getTestEmailAccountUrl(id)), {
      method: 'POST',
    });
  }

  async deleteEmailAccount(id: number): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getDeleteEmailAccountUrl(id)), {
      method: 'DELETE',
    });
  }

  async batchDeleteEmailAccounts(accountIds: number[]): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getBatchDeleteEmailAccountsUrl()), {
      method: 'POST',
      body: JSON.stringify({ account_ids: accountIds }),
    });
  }

  async batchSyncEmailAccounts(accountIds: number[]): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getBatchSyncEmailAccountsUrl()), {
      method: 'POST',
      body: JSON.stringify({ account_ids: accountIds }),
    });
  }

  async markAccountAsRead(accountId: number): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getMarkAccountAsReadUrl(accountId)), {
      method: 'PUT',
    });
  }

  async batchMarkAccountsAsRead(accountIds: number[]): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getBatchMarkAccountsAsReadUrl()), {
      method: 'POST',
      body: JSON.stringify({ account_ids: accountIds }),
    });
  }

  // 邮箱分组相关 API
  async getEmailGroups(): Promise<ApiResponse<EmailGroup[]>> {
    return this.request(this.generatedEndpoint(getListEmailGroupsUrl()));
  }

  async createEmailGroup(payload: { name: string }): Promise<ApiResponse<EmailGroup>> {
    return this.request<EmailGroup>(this.generatedEndpoint(getCreateEmailGroupUrl()), {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async updateEmailGroup(id: number, payload: { name?: string }): Promise<ApiResponse<EmailGroup>> {
    return this.request<EmailGroup>(this.generatedEndpoint(getUpdateEmailGroupUrl(id)), {
      method: 'PUT',
      body: JSON.stringify(payload),
    });
  }

  async deleteEmailGroup(id: number): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getDeleteEmailGroupUrl(id)), {
      method: 'DELETE',
    });
  }

  async setDefaultEmailGroup(id: number): Promise<ApiResponse<EmailGroup>> {
    return this.request<EmailGroup>(this.generatedEndpoint(getSetDefaultEmailGroupUrl(id)), {
      method: 'PUT',
    });
  }

  async reorderEmailGroups(groupIds: number[]): Promise<ApiResponse<EmailGroup[]>> {
    return this.request<EmailGroup[]>(this.generatedEndpoint(getReorderEmailGroupsUrl()), {
      method: 'PUT',
      body: JSON.stringify({ group_ids: groupIds }),
    });
  }

  // OAuth2 相关 API
  async getGmailOAuthUrl(
    callbackUrl?: string
  ): Promise<ApiResponse<{ auth_url: string; state: string }>> {
    return this.request(
      this.generatedEndpoint(getInitGmailOAuthUrl({ callback_url: callbackUrl }))
    );
  }

  async getOutlookOAuthUrl(
    callbackUrl?: string
  ): Promise<ApiResponse<{ auth_url: string; state: string }>> {
    return this.request(
      this.generatedEndpoint(getInitOutlookOAuthUrl({ callback_url: callbackUrl }))
    );
  }

  // 通过后端API处理OAuth2回调（后端会调用外部OAuth服务器）
  async handleOAuth2Callback(
    provider: string,
    code: string,
    state: string
  ): Promise<
    ApiResponse<{
      access_token: string;
      refresh_token?: string;
      token_type: string;
      expires_in: number;
      scope?: string;
    }>
  > {
    // 调用后端的OAuth回调处理端点
    return this.request(
      this.generatedEndpoint(
        getHandleOAuthCallbackUrl(provider as 'gmail' | 'outlook', { code, state })
      ),
      {
        method: 'GET',
      }
    );
  }

  async createOAuth2Account(account: {
    name: string;
    email: string;
    provider: string;
    access_token: string;
    refresh_token: string; // 必需，用于token验证和刷新
    expires_at: number;
    scope?: string;
    client_id: string; // 必需，用于token刷新
    group_id?: number;
  }): Promise<ApiResponse<EmailAccount>> {
    return this.request<EmailAccount>(this.generatedEndpoint(getCreateOAuthAccountUrl()), {
      method: 'POST',
      body: JSON.stringify(account),
    });
  }

  async createManualOAuth2Account(account: {
    name: string;
    email: string;
    provider: string;
    client_id: string;
    client_secret?: string;
    refresh_token: string;
    scope?: string;
    auth_url?: string;
    token_url?: string;
    group_id?: number;
  }): Promise<ApiResponse<EmailAccount>> {
    return this.request<EmailAccount>(this.generatedEndpoint(getCreateManualOAuthAccountUrl()), {
      method: 'POST',
      body: JSON.stringify(account),
    });
  }

  async createCustomEmailAccount(data: {
    name: string;
    email: string;
    auth_method: string;
    username: string;
    password: string;
    imap_host?: string;
    imap_port?: number;
    imap_security?: string;
    smtp_host?: string;
    smtp_port?: number;
    smtp_security?: string;
    group_id?: number;
  }): Promise<ApiResponse<EmailAccount>> {
    return this.request<EmailAccount>(this.generatedEndpoint(getCreateCustomEmailAccountUrl()), {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  // 邮件相关 API
  async getEmails(params?: {
    account_id?: number;
    folder_id?: number;
    page?: number;
    page_size?: number;
    search?: string;
    is_read?: boolean;
    is_starred?: boolean;
    is_important?: boolean;
    sort_by?: string;
    sort_order?: string;
  }): Promise<ApiResponse<{ emails: Email[]; total: number; page: number; page_size: number }>> {
    return this.request(this.generatedEndpoint(getListEmailsUrl(params as Parameters<typeof getListEmailsUrl>[0])));
  }

  async searchEmails(params: {
    q?: string; // 全文搜索关键词
    subject?: string; // 主题搜索
    from?: string; // 发件人搜索
    to?: string; // 收件人搜索
    body?: string; // 正文搜索
    since?: string; // 开始时间 (RFC3339格式)
    before?: string; // 结束时间 (RFC3339格式)
    has_attachment?: boolean; // 是否有附件
    is_read?: boolean; // 是否已读
    is_starred?: boolean; // 是否加星
    account_id?: number; // 账户ID筛选
    folder_id?: number; // 文件夹ID筛选
    page?: number; // 页码
    page_size?: number; // 每页大小
  }): Promise<
    ApiResponse<{
      emails: Email[];
      total: number;
      page: number;
      page_size: number;
      total_pages?: number;
    }>
  > {
    console.log('🌐 [ApiClient] searchEmails() 被调用:', params);

    const url = this.generatedEndpoint(getSearchEmailsUrl(params));
    console.log('🌐 [ApiClient] 请求URL:', url);

    try {
      const result = await this.request<{
        emails: Email[];
        total: number;
        page: number;
        page_size: number;
        total_pages?: number;
      }>(url);
      console.log('🌐 [ApiClient] searchEmails 响应:', {
        success: result.success,
        dataExists: !!result.data,
        emailCount: result.data?.emails?.length || 0,
        total: result.data?.total || 0,
        message: result.message,
      });
      return result;
    } catch (error) {
      console.error('🌐 [ApiClient] searchEmails 错误:', error);
      throw error;
    }
  }

  async getFolders(accountId?: number): Promise<ApiResponse<Folder[]>> {
    return this.request(this.generatedEndpoint(getListFoldersUrl({ account_id: accountId })));
  }

  async getFolder(folderId: number): Promise<ApiResponse<Folder>> {
    return this.request(this.generatedEndpoint(getGetFolderUrl(folderId)));
  }

  async getEmailStats(): Promise<ApiResponse<EmailStats>> {
    return this.request('/emails/stats');
  }

  async batchEmailOperation(data: {
    email_ids: number[];
    operation: string;
    target_folder_id?: number;
  }): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getBatchEmailOperationsUrl()), {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async markAllAsRead(folderId: number): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getMarkFolderAsReadUrl(folderId)), {
      method: 'PUT',
    });
  }

  async moveEmail(emailId: number, targetFolderId: number): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getMoveEmailUrl(emailId)), {
      method: 'PUT',
      body: JSON.stringify({ target_folder_id: targetFolderId }),
    });
  }

  async createFolder(data: {
    account_id: number;
    name: string;
    display_name?: string;
    parent_id?: number;
  }): Promise<ApiResponse<Folder>> {
    return this.request(this.generatedEndpoint(getCreateFolderUrl()), {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateFolder(
    folderId: number,
    data: {
      name?: string;
      display_name?: string;
    }
  ): Promise<ApiResponse<Folder>> {
    return this.request(this.generatedEndpoint(getUpdateFolderUrl(folderId)), {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteFolder(folderId: number): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getDeleteFolderUrl(folderId)), {
      method: 'DELETE',
    });
  }

  async markFolderAsRead(folderId: number): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getMarkFolderAsReadUrl(folderId)), {
      method: 'PUT',
    });
  }

  async syncFolder(folderId: number): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getSyncFolderUrl(folderId)), {
      method: 'PUT',
    });
  }

  async syncAccount(accountId: number): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getSyncEmailAccountUrl(accountId)), {
      method: 'POST',
    });
  }

  async getEmailDetail(emailId: number): Promise<ApiResponse<Email>> {
    return this.request(this.generatedEndpoint(getGetEmailUrl(emailId)));
  }

  async markEmailAsRead(emailId: number): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getMarkEmailAsReadUrl(emailId)), {
      method: 'PUT',
    });
  }

  async markEmailAsUnread(emailId: number): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getMarkEmailAsUnreadUrl(emailId)), {
      method: 'PUT',
    });
  }

  async toggleEmailStar(emailId: number): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getToggleEmailStarUrl(emailId)), {
      method: 'PUT',
    });
  }

  async deleteEmail(emailId: number): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getDeleteEmailUrl(emailId)), {
      method: 'DELETE',
    });
  }

  async toggleEmailImportant(emailId: number): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getUpdateEmailUrl(emailId)), {
      method: 'PATCH',
      body: JSON.stringify({ is_important: true }), // 后端会自动切换状态
    });
  }

  async downloadAttachment(attachmentId: number): Promise<Blob> {
    const response = await fetch(
      `${API_BASE_URL}${this.generatedEndpoint(getDownloadAttachmentUrl(attachmentId))}`,
      {
        headers: {
          Authorization: `Bearer ${this.getAuthToken()}`,
        },
      }
    );

    if (!response.ok) {
      throw new Error('Failed to download attachment');
    }

    return response.blob();
  }

  async getEmail(id: number): Promise<ApiResponse<Email>> {
    return this.request(this.generatedEndpoint(getGetEmailUrl(id)));
  }

  async sendEmail(email: {
    account_id: number;
    to: EmailAddress[];
    cc?: EmailAddress[];
    bcc?: EmailAddress[];
    subject: string;
    text_body?: string;
    html_body?: string;
    attachment_ids?: number[];
    priority?: string;
    importance?: string;
    scheduled_time?: string;
    request_read_receipt?: boolean;
    request_delivery_receipt?: boolean;
  }): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getSendEmailUrl()), {
      method: 'POST',
      body: JSON.stringify(email),
    });
  }

  async sendBulkEmails(payload: {
    account_id: number;
    emails: Array<Record<string, unknown>>;
  }): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getSendBulkEmailsUrl()), {
      method: 'POST',
      body: JSON.stringify(payload),
    });
  }

  async getSendStatus(sendId: string): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getGetSendStatusUrl(sendId)));
  }

  async resendEmail(sendId: string): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getResendEmailUrl(sendId)), {
      method: 'POST',
    });
  }

  async replyEmail(
    originalEmailId: number,
    email: {
      account_id: number;
      to: EmailAddress[];
      cc?: EmailAddress[];
      bcc?: EmailAddress[];
      subject: string;
      text_body?: string;
      html_body?: string;
      attachment_ids?: number[];
      priority?: string;
      importance?: string;
      scheduled_time?: string;
      request_read_receipt?: boolean;
      request_delivery_receipt?: boolean;
    }
  ): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getReplyEmailUrl(originalEmailId)), {
      method: 'POST',
      body: JSON.stringify(email),
    });
  }

  async replyAllEmail(
    originalEmailId: number,
    email: {
      account_id: number;
      to: EmailAddress[];
      cc?: EmailAddress[];
      bcc?: EmailAddress[];
      subject: string;
      text_body?: string;
      html_body?: string;
      attachment_ids?: number[];
      priority?: string;
      importance?: string;
      scheduled_time?: string;
      request_read_receipt?: boolean;
      request_delivery_receipt?: boolean;
    }
  ): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getReplyAllEmailUrl(originalEmailId)), {
      method: 'POST',
      body: JSON.stringify(email),
    });
  }

  async forwardEmail(
    originalEmailId: number,
    email: {
      account_id: number;
      to: EmailAddress[];
      cc?: EmailAddress[];
      bcc?: EmailAddress[];
      subject: string;
      text_body?: string;
      html_body?: string;
      priority?: string;
      importance?: string;
      scheduled_time?: string;
      request_read_receipt?: boolean;
      request_delivery_receipt?: boolean;
    }
  ): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getForwardEmailUrl(originalEmailId)), {
      method: 'POST',
      body: JSON.stringify(email),
    });
  }

  async archiveEmail(emailId: number): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getArchiveEmailUrl(emailId)), {
      method: 'PUT',
    });
  }

  async saveDraft(draft: {
    accountId?: number;
    to: string[];
    cc?: string[];
    bcc?: string[];
    subject: string;
    content?: string;
    htmlContent?: string;
    attachments?: unknown[];
  }): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getSaveDraftUrl()), {
      method: 'POST',
      body: JSON.stringify({
        account_id: draft.accountId,
        to: draft.to.map((email) => ({ address: email })),
        cc: draft.cc?.map((email) => ({ address: email })) || [],
        bcc: draft.bcc?.map((email) => ({ address: email })) || [],
        subject: draft.subject,
        text_body: draft.content,
        html_body: draft.htmlContent,
      }),
    });
  }

  async getDraft(id: number): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getGetDraftUrl(id)));
  }

  async updateDraft(id: number, draft: Record<string, unknown>): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getUpdateDraftUrl(id)), {
      method: 'PUT',
      body: JSON.stringify(draft),
    });
  }

  async deleteDraft(id: number): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getDeleteDraftUrl(id)), {
      method: 'DELETE',
    });
  }

  async listDrafts(params?: { page?: number; page_size?: number }): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getListDraftsUrl(params)));
  }

  async createTemplate(template: Record<string, unknown>): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getCreateTemplateUrl()), {
      method: 'POST',
      body: JSON.stringify(template),
    });
  }

  async getTemplate(id: number): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getGetTemplateUrl(id)));
  }

  async updateTemplate(id: number, template: Record<string, unknown>): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getUpdateTemplateUrl(id)), {
      method: 'PUT',
      body: JSON.stringify(template),
    });
  }

  async deleteTemplate(id: number): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getDeleteTemplateUrl(id)), {
      method: 'DELETE',
    });
  }

  async listTemplates(params?: {
    page?: number;
    page_size?: number;
    category?: string;
    search?: string;
  }): Promise<ApiResponse> {
    return this.request(this.generatedEndpoint(getListTemplatesUrl(params)));
  }
}

export const apiClient = new ApiClient();

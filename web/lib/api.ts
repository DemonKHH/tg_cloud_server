/**
 * API服务封装
 * 统一处理API请求，包含认证、错误处理等
 */

// 开发环境使用相对路径（通过 Next.js rewrites 代理）
// 生产环境可以使用完整 URL 或继续使用代理
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || '/api/v1';

import { toast } from "sonner";

export interface APIResponse<T = any> {
  code: number;
  msg: string;
  data?: T;
}

export interface PaginationResponse<T> {
  data: T[];
  total: number;
  page: number;
  limit: number;
  total_pages: number;
  has_next: boolean;
  has_previous: boolean;
}

// 响应码常量
export const ResponseCode = {
  SUCCESS: 0,
  INVALID_PARAM: 1001,
  UNAUTHORIZED: 1002,
  FORBIDDEN: 1003,
  NOT_FOUND: 1004,
  INTERNAL_ERROR: 1005,
  RATE_LIMIT: 1006,
  CONFLICT: 1007,
} as const;

class ApiClient {
  private baseURL: string;
  private token: string | null = null;

  constructor(baseURL: string) {
    this.baseURL = baseURL;
    if (typeof window !== 'undefined') {
      this.token = localStorage.getItem('token');
    }
  }

  setToken(token: string | null) {
    this.token = token;
    if (token && typeof window !== 'undefined') {
      localStorage.setItem('token', token);
    } else if (typeof window !== 'undefined') {
      localStorage.removeItem('token');
    }
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<APIResponse<T>> {
    const url = `${this.baseURL}${endpoint}`;
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(options.headers as Record<string, string>),
    };

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    try {
      const response = await fetch(url, {
        ...options,
        headers,
      });

      // 尝试解析 JSON
      let data: APIResponse<T>;
      try {
        data = await response.json();
      } catch (jsonError) {
        // 如果无法解析 JSON，可能是网络错误或其他非 API 响应
        if (!response.ok) {
          throw new Error(`请求失败: ${response.status} ${response.statusText}`);
        }
        throw new Error('无效的响应格式');
      }

      // 优先检查业务状态码
      if (data.code !== ResponseCode.SUCCESS) {
        // 专门处理未授权
        if (data.code === ResponseCode.UNAUTHORIZED ||
          data.code === 40101 || // 兼容 errors.ErrCodeUnauthorized
          data.code === 40103 || // TokenExpired
          data.code === 40104    // TokenInvalid
        ) {
          this.handleUnauthorized();
        }
        // 不再抛出错误，也不显示Toast，直接返回数据让业务层处理
      }

      return data;
    } catch (error) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error('网络请求失败');
    }
  }

  private handleUnauthorized() {
    // 清除 token
    this.setToken(null);
    // 重定向到登录页面
    if (typeof window !== 'undefined') {
      // 保存当前路径，登录后可以跳转回来
      const currentPath = window.location.pathname;
      if (currentPath !== '/login') {
        localStorage.setItem('redirectAfterLogin', currentPath);
        window.location.href = '/login';
      }
    }
  }

  async get<T>(endpoint: string, params?: Record<string, any>): Promise<APIResponse<T>> {
    let url = endpoint;
    if (params) {
      const searchParams = new URLSearchParams();
      Object.entries(params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          searchParams.append(key, String(value));
        }
      });
      url += `?${searchParams.toString()}`;
    }
    return this.request<T>(url, { method: 'GET' });
  }

  async post<T>(endpoint: string, data?: any): Promise<APIResponse<T>> {
    return this.request<T>(endpoint, {
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async put<T>(endpoint: string, data?: any): Promise<APIResponse<T>> {
    return this.request<T>(endpoint, {
      method: 'PUT',
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  async delete<T>(endpoint: string): Promise<APIResponse<T>> {
    return this.request<T>(endpoint, { method: 'DELETE' });
  }

  async postFormData<T>(endpoint: string, formData: FormData): Promise<APIResponse<T>> {
    const url = `${this.baseURL}${endpoint}`;
    const headers: Record<string, string> = {};

    if (this.token) {
      headers['Authorization'] = `Bearer ${this.token}`;
    }

    const response = await fetch(url, {
      method: 'POST',
      headers,
      body: formData,
    });

    // 移除 HTTP 状态检查，完全依赖 JSON 响应
    // 但如果非 200 OK 且不能解析 JSON，可能需要处理（此处假设 API 总是返回 JSON）

    const data: APIResponse<T> = await response.json();

    // 优先检查业务状态码
    if (data.code !== ResponseCode.SUCCESS) {
      // 专门处理未授权
      if (data.code === ResponseCode.UNAUTHORIZED ||
        data.code === 40101 || // 兼容 errors.ErrCodeUnauthorized
        data.code === 40103 || // TokenExpired
        data.code === 40104    // TokenInvalid
      ) {
        this.handleUnauthorized();
      }
      throw new Error(data.msg || '请求失败');
    }
    return data;
  }
}

export const apiClient = new ApiClient(API_BASE_URL);

// 认证相关API
export const authAPI = {
  login: (username: string, password: string) =>
    apiClient.post('/auth/login', { username, password }),
  register: (data: { username: string; password: string; email: string }) =>
    apiClient.post('/auth/register', data),
  logout: () => apiClient.post('/auth/logout'),
  getProfile: () => apiClient.get('/auth/profile'),
  updateProfile: (data: any) => apiClient.post('/auth/profile', data),
  refresh: () => apiClient.post('/auth/refresh'),
};

// 账号管理API
export const accountAPI = {
  list: (params?: { page?: number; limit?: number; status?: string }) =>
    apiClient.get<PaginationResponse<any>>('/accounts', params),
  get: (id: string) => apiClient.get(`/accounts/${id}`),
  create: (data: any) => apiClient.post('/accounts', data),
  update: (id: string, data: any) => apiClient.post(`/accounts/${id}/update`, data),
  delete: (id: string) => apiClient.post(`/accounts/${id}/delete`),
  checkHealth: (id: string) => apiClient.get(`/accounts/${id}/health`),
  getAvailability: (id: string) => apiClient.get(`/accounts/${id}/availability`),
  bindProxy: (id: string, proxyId?: number) =>
    apiClient.post(`/accounts/${id}/bind-proxy`, { proxy_id: proxyId || null }),
  uploadFiles: (file: File, proxyId?: number) => {
    const formData = new FormData();
    formData.append('file', file);
    if (proxyId) {
      formData.append('proxy_id', proxyId.toString());
    }
    return apiClient.postFormData('/accounts/upload', formData);
  },
  getQueueInfo: (id: string) => apiClient.get(`/accounts/${id}/queue`),
  batchBindProxy: (accountIds: string[], proxyId?: number) =>
    apiClient.post('/accounts/batch/bind-proxy', { account_ids: accountIds.map(Number), proxy_id: proxyId || null }),
  batchSet2FA: (accountIds: string[], password: string) =>
    apiClient.post('/accounts/batch/set-2fa', { account_ids: accountIds.map(Number), password }),
  batchUpdate2FA: (accountIds: string[], newPassword: string, oldPassword?: string) =>
    apiClient.post('/accounts/batch/update-2fa', { account_ids: accountIds.map(Number), new_password: newPassword, old_password: oldPassword }),
  batchDelete: (accountIds: string[]) =>
    apiClient.post('/accounts/batch/delete', { account_ids: accountIds.map(Number) }),
  export: async (accountIds: string[]) => {
    const url = `${process.env.NEXT_PUBLIC_API_URL || '/api/v1'}/accounts/export`;
    const token = typeof window !== 'undefined' ? localStorage.getItem('token') : null;
    const response = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        ...(token ? { 'Authorization': `Bearer ${token}` } : {}),
      },
      body: JSON.stringify({ account_ids: accountIds.map(Number) }),
    });
    if (!response.ok) {
      const data = await response.json();
      throw new Error(data.msg || '导出失败');
    }
    return response.blob();
  },
};

// 任务管理API
export const taskAPI = {
  list: (params?: { page?: number; limit?: number; status?: string; account_id?: string }) =>
    apiClient.get<PaginationResponse<any>>('/tasks', params),
  get: (id: string) => apiClient.get(`/tasks/${id}`),
  create: (data: any) => apiClient.post('/tasks', data),
  update: (id: string, data: any) => apiClient.post(`/tasks/${id}/update`, data),
  delete: (id: string) => apiClient.post(`/tasks/${id}/delete`),
  cancel: (id: string) => apiClient.post(`/tasks/${id}/cancel`),
  retry: (id: string) => apiClient.post(`/tasks/${id}/retry`),
  control: (id: string, action: 'start' | 'pause' | 'stop' | 'resume') =>
    apiClient.post(`/tasks/${id}/control`, { action }),
  batchControl: (ids: string[], action: 'start' | 'pause' | 'stop' | 'resume' | 'cancel') =>
    apiClient.post('/tasks/batch/control', { task_ids: ids, action }),
  getLogs: (id: string) => apiClient.get(`/tasks/${id}/logs`),
  getStats: () => apiClient.get('/tasks/stats'),
  batchCancel: (ids: string[]) => apiClient.post('/tasks/batch/cancel', { task_ids: ids }),
  batchDelete: (ids: string[]) => apiClient.post('/tasks/batch/delete', { task_ids: ids }),
};

// 代理管理API
export const proxyAPI = {
  list: (params?: { page?: number; limit?: number; status?: string }) =>
    apiClient.get<PaginationResponse<any>>('/proxies', params),
  get: (id: string) => apiClient.get(`/proxies/${id}`),
  create: (data: any) => apiClient.post('/proxies', data),
  batchCreate: (data: { proxies: any[] }) => apiClient.post('/proxies/batch', data),
  batchDelete: (ids: number[]) => apiClient.post('/proxies/batch/delete', { proxy_ids: ids }),
  batchTest: (ids: number[]) => apiClient.post('/proxies/batch/test', { proxy_ids: ids }),
  update: (id: string, data: any) => apiClient.post(`/proxies/${id}/update`, data),
  delete: (id: string) => apiClient.post(`/proxies/${id}/delete`),
  test: (id: string) => apiClient.post(`/proxies/${id}/test`),
  getStats: () => apiClient.get('/proxies/stats'),
};

// 模块功能API
export const moduleAPI = {
  accountCheck: (data: { account_id: string;[key: string]: any }) =>
    apiClient.post('/modules/check', data),
  privateMessage: (data: any) => apiClient.post('/modules/private', data),
  broadcast: (data: any) => apiClient.post('/modules/broadcast', data),
  verifyCode: (data: any) => apiClient.post('/modules/verify', data),
  groupChat: (data: any) => apiClient.post('/modules/groupchat', data),
};

// 文件管理API
export const fileAPI = {
  list: (params?: any) => apiClient.get<PaginationResponse<any>>('/files', params),
  get: (id: string) => apiClient.get(`/files/${id}`),
  upload: (file: File, category: string) => {
    const formData = new FormData();
    formData.append('file', file);
    formData.append('category', category);
    return apiClient.postFormData('/files/upload', formData);
  },
  uploadFromURL: (url: string, category: string) =>
    apiClient.post('/files/upload-url', { url, category }),
  delete: (id: string) => apiClient.post(`/files/${id}/delete`),
  download: (id: string) => `${API_BASE_URL}/files/${id}/download`,
  preview: (id: string) => `${API_BASE_URL}/files/${id}/preview`,
  getURL: (id: string) => apiClient.get(`/files/${id}/url`),
  batchUpload: (files: File[], category: string) => {
    const formData = new FormData();
    files.forEach((file) => formData.append('files', file));
    formData.append('category', category);
    return apiClient.postFormData('/files/batch-upload', formData);
  },
  batchDelete: (ids: string[]) =>
    apiClient.post('/files/batch-delete', { file_ids: ids.map(Number) }),
};

// AI服务API
export const aiAPI = {
  generateGroupChat: (config: any) => apiClient.post('/ai/group-chat', config),
  generatePrivateMessage: (config: any) => apiClient.post('/ai/private-message', config),
  analyzeSentiment: (text: string) => apiClient.post('/ai/analyze-sentiment', { text }),
  extractKeywords: (text: string) => apiClient.post('/ai/extract-keywords', { text }),
  generateVariations: (template: string, count: number) =>
    apiClient.post('/ai/generate-variations', { template, count }),
  getConfig: () => apiClient.get('/ai/config'),
  test: () => apiClient.post('/ai/test'),
};

// 验证码API
export const verifyCodeAPI = {
  // 生成验证码访问链接（需要认证）
  generate: (data: { account_id: number; expires_in?: number }) =>
    apiClient.post<{
      code: string;
      url: string;
      expires_at: string;
      expires_in: number;
    }>('/verify-code/generate', data),
  // 批量生成验证码访问链接
  batchGenerate: (data: { account_ids: number[]; expires_in?: number }) =>
    apiClient.post<{
      items: {
        account_id: number;
        phone: string;
        code: string;
        url: string;
        expires_at: number;
        expires_in: number;
      }[];
    }>('/verify-code/batch/generate', data),
  // 获取验证码（公开接口，无需认证）
  getCode: (code: string, timeout?: number) => {
    const params = timeout ? { timeout } : {};
    return apiClient.get<{
      success: boolean;
      code?: string;
      sender?: string;
      received_at?: string;
      wait_seconds?: number;
      message: string;
    }>(`/verify-code/${code}`, params);
  },
  // 获取会话信息（调试用，需要认证）
  getSessionInfo: (code: string) => apiClient.get(`/verify-code/${code}/info`),
  // 获取会话列表
  listSessions: (params?: { page?: number; limit?: number; keyword?: string }) => apiClient.get<{
    items: {
      code: string;
      url: string;
      account_id: number;
      account_phone: string;
      expires_at: number;
      expires_in: number;
      created_at: number;
    }[];
    pagination: {
      total: number;
      page: number;
      limit: number;
      pages: number;
    };
  }>('/verify-code/sessions', params),
  // 批量删除会话
  batchDelete: (codes: string[]) => apiClient.post('/verify-code/batch/delete', { codes }),
};

// 统计API
export const statsAPI = {
  getOverview: (period?: string) => apiClient.get('/stats/overview', { period }),
  getAccountStats: (period?: string, status?: string) =>
    apiClient.get('/stats/accounts', { period, status }),
  getDashboard: () => apiClient.get('/stats/dashboard'),
  getTaskStats: () => apiClient.get('/stats/tasks'),
  getProxyStats: () => apiClient.get('/stats/proxies'),
};

// 设置API
export interface RiskSettings {
  max_consecutive_failures: number;
  cooling_duration_minutes: number;
}

export const settingsAPI = {
  getRiskSettings: () => apiClient.get<RiskSettings>('/settings/risk'),
  updateRiskSettings: (data: RiskSettings) =>
    apiClient.put<RiskSettings>('/settings/risk', data),
};

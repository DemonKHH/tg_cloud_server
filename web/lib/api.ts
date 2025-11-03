/**
 * API服务封装
 * 统一处理API请求，包含认证、错误处理等
 */

// 开发环境使用相对路径（通过 Next.js rewrites 代理）
// 生产环境可以使用完整 URL 或继续使用代理
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || '/api/v1';

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

      const data: APIResponse<T> = await response.json();

      if (data.code !== ResponseCode.SUCCESS) {
        throw new Error(data.msg || '请求失败');
      }

      return data;
    } catch (error) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error('网络请求失败');
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

    const data: APIResponse<T> = await response.json();
    if (data.code !== ResponseCode.SUCCESS) {
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
    apiClient.post('/accounts/batch/bind-proxy', { account_ids: accountIds, proxy_id: proxyId || null }),
};

// 任务管理API
export const taskAPI = {
  list: (params?: { page?: number; limit?: number; status?: string; account_id?: string }) =>
    apiClient.get<PaginationResponse<any>>('/tasks', params),
  get: (id: string) => apiClient.get(`/tasks/${id}`),
  create: (data: any) => apiClient.post('/tasks', data),
  update: (id: string, data: any) => apiClient.post(`/tasks/${id}/update`, data),
  cancel: (id: string) => apiClient.post(`/tasks/${id}/cancel`),
  retry: (id: string) => apiClient.post(`/tasks/${id}/retry`),
  getLogs: (id: string) => apiClient.get(`/tasks/${id}/logs`),
  getStats: () => apiClient.get('/tasks/stats'),
  batchCancel: (ids: string[]) => apiClient.post('/tasks/batch/cancel', { task_ids: ids }),
};

// 代理管理API
export const proxyAPI = {
  list: (params?: { page?: number; limit?: number; status?: string }) =>
    apiClient.get<PaginationResponse<any>>('/proxies', params),
  get: (id: string) => apiClient.get(`/proxies/${id}`),
  create: (data: any) => apiClient.post('/proxies', data),
  update: (id: string, data: any) => apiClient.post(`/proxies/${id}/update`, data),
  delete: (id: string) => apiClient.post(`/proxies/${id}/delete`),
  test: (id: string) => apiClient.post(`/proxies/${id}/test`),
  getStats: () => apiClient.get('/proxies/stats'),
};

// 模块功能API
export const moduleAPI = {
  accountCheck: (data: { account_id: string; [key: string]: any }) =>
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

// 统计API
export const statsAPI = {
  getOverview: (period?: string) => apiClient.get('/stats/overview', { period }),
  getAccountStats: (period?: string, status?: string) =>
    apiClient.get('/stats/accounts', { period, status }),
  getDashboard: () => apiClient.get('/stats/dashboard'),
  getTaskStats: () => apiClient.get('/stats/tasks'),
  getProxyStats: () => apiClient.get('/stats/proxies'),
};


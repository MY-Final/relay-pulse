/** 管理员可复用的代理配置。代理地址只返回脱敏版本。 */
export interface ProxyProfile {
  id: string;
  name: string;
  url_masked: string;
  created_at?: string;
  updated_at?: string;
}


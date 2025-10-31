import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // 代理配置：将 /api/v1/* 代理到后端服务器
  async rewrites() {
    const backendUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
    
    return [
      {
        source: '/api/v1/:path*',
        destination: `${backendUrl}/api/v1/:path*`,
      },
    ];
  },
};

export default nextConfig;

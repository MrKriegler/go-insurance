/** @type {import('next').NextConfig} */
const nextConfig = {
  // Static export for S3/CloudFront deployment
  output: 'export',
  trailingSlash: true,
  images: {
    unoptimized: true,
  },
  // Rewrites only work in dev mode (not static export)
  async rewrites() {
    // Only apply in development (these don't work with static export)
    if (process.env.NODE_ENV === 'development') {
      return [
        {
          source: '/api/v1/:path*',
          destination: 'http://localhost:8080/api/v1/:path*',
        },
      ];
    }
    return [];
  },
};

module.exports = nextConfig;

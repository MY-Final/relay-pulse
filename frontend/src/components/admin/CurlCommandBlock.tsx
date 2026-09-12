import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Copy, Check } from 'lucide-react';
import { copyToClipboard } from '../../utils/share';

interface CurlCommandBlockProps {
  /** 后端下发的脱敏 curl（密钥用 $RP_API_KEY 占位）。空串时不渲染。 */
  curl: string;
}

/**
 * 管理员测试结果里展示「本次实际请求」对应的可复制 curl 命令。
 *
 * 安全：这里只提供脱敏版本（密钥用 $RP_API_KEY 占位），避免真实密钥进入
 * 浏览器状态、响应正文或剪贴板。
 */
export function CurlCommandBlock({ curl }: CurlCommandBlockProps) {
  const { t } = useTranslation();
  const [copied, setCopied] = useState(false);

  if (!curl) return null;

  const usesKeyVar = curl.includes('$RP_API_KEY');

  const doCopy = async () => {
    if (await copyToClipboard(curl)) {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  return (
    <div className="space-y-1">
      <div className="flex flex-wrap items-center gap-3">
        <span className="text-xs text-muted">{t('admin.detail.testCurl')}</span>
        <button
          type="button"
          onClick={doCopy}
          className="inline-flex items-center gap-1 text-xs text-secondary hover:text-primary transition"
        >
          {copied ? <Check className="w-3.5 h-3.5 text-success" /> : <Copy className="w-3.5 h-3.5" />}
          {t('admin.detail.testCurlCopy')}
        </button>
      </div>
      <pre className="whitespace-pre-wrap break-all text-xs max-h-48 overflow-y-auto bg-surface p-2 rounded text-secondary font-mono">
        {curl}
      </pre>
      {usesKeyVar && (
        <p className="text-[11px] text-muted">{t('admin.detail.testCurlKeyHint')}</p>
      )}
    </div>
  );
}

import React, { useState } from 'react';
import { filesApi } from '../../api/files';
import { FileText, Download, Loader2, Image as ImageIcon } from 'lucide-react';
import { formatBytes } from '../../lib/utils';

interface AttachmentCardProps {
  fileId: string;
  name?: string;
  size?: number;
  mimeType?: string;
}

export const AttachmentCard: React.FC<AttachmentCardProps> = ({
  fileId,
  name = 'Attachment',
  size,
  mimeType,
}) => {
  const [downloading, setDownloading] = useState(false);
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);

  const isImage = mimeType?.startsWith('image/') || /\.(png|jpe?g|gif|webp|svg)$/i.test(name);

  const handleDownload = async () => {
    setDownloading(true);
    try {
      const { presigned_url } = await filesApi.getDownloadUrl(fileId);
      window.open(presigned_url, '_blank', 'noopener,noreferrer');
    } catch (e) {
      console.error('Failed to get presigned download URL', e);
    } finally {
      setDownloading(false);
    }
  };

  const handleLoadImagePreview = async () => {
    if (!previewUrl && isImage) {
      try {
        const { presigned_url } = await filesApi.getDownloadUrl(fileId);
        setPreviewUrl(presigned_url);
      } catch (e) {
        console.error('Failed to load image preview', e);
      }
    }
  };

  return (
    <div className="mt-2 max-w-sm rounded border border-brand-border bg-brand-surface p-2.5 space-y-2">
      {isImage && previewUrl ? (
        <div className="max-h-60 overflow-hidden rounded border border-brand-border bg-brand-black">
          <img src={previewUrl} alt={name} className="object-contain w-full h-full" />
        </div>
      ) : null}

      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-2.5 overflow-hidden">
          <div className="w-8 h-8 rounded bg-brand-card border border-brand-border flex items-center justify-center text-brand-amber shrink-0">
            {isImage ? <ImageIcon className="w-4 h-4" /> : <FileText className="w-4 h-4" />}
          </div>
          <div className="flex flex-col min-w-0">
            <span className="text-xs font-mono text-brand-textHeading truncate">{name}</span>
            <span className="text-[10px] font-mono text-zinc-500">
              {size ? formatBytes(size) : 'File attachment'}
            </span>
          </div>
        </div>

        <div className="flex items-center gap-1">
          {isImage && !previewUrl && (
            <button
              onClick={handleLoadImagePreview}
              className="px-2 py-1 text-[11px] font-mono rounded bg-brand-card text-brand-textBody hover:text-brand-textHeading border border-brand-border hover:bg-zinc-800 transition-colors"
            >
              Preview
            </button>
          )}

          <button
            onClick={handleDownload}
            disabled={downloading}
            className="p-1.5 rounded text-brand-textMuted hover:text-brand-amber hover:bg-brand-card transition-colors shrink-0"
            title="Download File"
          >
            {downloading ? (
              <Loader2 className="w-4 h-4 animate-spin text-brand-amber" />
            ) : (
              <Download className="w-4 h-4" />
            )}
          </button>
        </div>
      </div>
    </div>
  );
};

'use client'

import { useState, useRef } from 'react'
import { Button } from '@/components/ui/button'
import { Upload, X } from 'lucide-react'
import { useToast } from '@/components/ui/use-toast'

interface ReferenceImageUploadProps {
  preview: string | null
  onUpload: (file: File) => void
  onRemove: () => void
  className?: string
}

export function ReferenceImageUpload({
  preview,
  onUpload,
  onRemove,
  className = '',
}: ReferenceImageUploadProps) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [isDragging, setIsDragging] = useState(false)
  const { toast } = useToast()

  const handleFileSelect = (selectedFile: File) => {
    // 验证文件类型
    if (!selectedFile.type.startsWith('image/')) {
      toast({
        title: '文件格式不支持',
        description: '请选择图片文件（JPG/PNG/WebP 等）',
        variant: 'destructive',
      })
      return
    }
    
    // 验证文件大小（最大 10MB）
    if (selectedFile.size > 10 * 1024 * 1024) {
      toast({
        title: '文件过大',
        description: '图片大小不能超过 10MB',
        variant: 'destructive',
      })
      return
    }

    onUpload(selectedFile)
  }

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault()
    setIsDragging(false)
    
    const droppedFile = e.dataTransfer.files[0]
    if (droppedFile) {
      handleFileSelect(droppedFile)
    }
  }

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault()
    setIsDragging(true)
  }

  const handleDragLeave = () => {
    setIsDragging(false)
  }

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const selectedFile = e.target.files?.[0]
    if (selectedFile) {
      handleFileSelect(selectedFile)
    }
  }

  const handleClick = () => {
    inputRef.current?.click()
  }

  if (preview) {
    return (
      <div className={`relative group ${className}`}>
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img
          src={preview}
          alt="参考图片"
          className="w-full h-48 object-cover rounded-lg border border-border"
        />
        <div className="absolute inset-0 bg-black/50 opacity-0 group-hover:opacity-100 transition-opacity rounded-lg flex items-center justify-center">
          <Button
            variant="destructive"
            size="sm"
            onClick={onRemove}
          >
            <X className="w-4 h-4 mr-2" />
            移除
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div
      className={`border-2 border-dashed rounded-lg p-6 text-center cursor-pointer transition-colors ${
        isDragging
          ? 'border-primary bg-primary/5'
          : 'border-border hover:border-primary/50 hover:bg-accent/50'
      } ${className}`}
      onDrop={handleDrop}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onClick={handleClick}
    >
      <input
        ref={inputRef}
        type="file"
        accept="image/*"
        onChange={handleInputChange}
        className="hidden"
      />
      <div className="flex flex-col items-center gap-3">
        <div className="w-12 h-12 rounded-full bg-primary/10 flex items-center justify-center">
          <Upload className="w-6 h-6 text-primary" />
        </div>
        <div>
          <p className="text-sm font-medium">点击或拖拽上传参考图</p>
          <p className="text-xs text-muted-foreground mt-1">
            支持 JPG、PNG、WebP 格式，最大 10MB
          </p>
        </div>
      </div>
    </div>
  )
}

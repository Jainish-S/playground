"use client";

import { useRef } from "react";
import { QRCodeCanvas } from "qrcode.react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Download, QrCode } from "lucide-react";
import { useToast } from "@/hooks/use-toast";

interface QRCodeDisplayProps {
  url: string;
  size?: number;
  fileName?: string;
}

export function QRCodeDisplay({ url, size = 200, fileName = "qr-code" }: QRCodeDisplayProps) {
  const qrRef = useRef<HTMLDivElement>(null);
  const { toast } = useToast();

  const downloadQRCode = () => {
    try {
      const canvas = qrRef.current?.querySelector("canvas");
      if (!canvas) {
        throw new Error("QR code canvas not found");
      }

      // Convert canvas to blob and download
      canvas.toBlob((blob) => {
        if (!blob) {
          throw new Error("Failed to generate QR code image");
        }

        const url = URL.createObjectURL(blob);
        const link = document.createElement("a");
        link.href = url;
        link.download = `${fileName}.png`;
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);
        URL.revokeObjectURL(url);

        toast({
          description: "QR code downloaded successfully!",
          duration: 3000,
        });
      });
    } catch (error) {
      console.error("Failed to download QR code:", error);
      toast({
        variant: "destructive",
        description: "Failed to download QR code",
        duration: 3000,
      });
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <QrCode className="h-5 w-5" />
          QR Code
        </CardTitle>
        <CardDescription>Scan to access this short URL</CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col items-center gap-4">
        <div
          ref={qrRef}
          className="p-4 bg-white rounded-lg border-2 border-border"
        >
          <QRCodeCanvas
            value={url}
            size={size}
            level="H"
            includeMargin={false}

          />
        </div>
        <Button
          onClick={downloadQRCode}
          variant="outline"
          size="sm"
          className="w-full"
        >
          <Download className="h-4 w-4 mr-2" />
          Download QR Code
        </Button>
      </CardContent>
    </Card>
  );
}

"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Loader2, CheckCircle2, XCircle, ChevronDown, ChevronUp } from "lucide-react";
import { useToast } from "@/hooks/use-toast";
import { checkCodeAction, createURLAction } from "@/app/actions";

interface CreateURLModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: () => void;
}

export function CreateURLModal({
  open,
  onOpenChange,
  onSuccess,
}: CreateURLModalProps) {
  const router = useRouter();
  const { toast } = useToast();

  const [destinationUrl, setDestinationUrl] = useState("");
  const [customCode, setCustomCode] = useState("");
  const [notes, setNotes] = useState("");
  const [showAdvanced, setShowAdvanced] = useState(false);

  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [isCheckingCode, setIsCheckingCode] = useState(false);
  const [codeAvailable, setCodeAvailable] = useState<boolean | null>(null);

  // Check if custom code is available (debounced)
  useEffect(() => {
    if (!customCode) {
      setCodeAvailable(null);
      return;
    }

    if (customCode.length < 4) {
      setCodeAvailable(null);
      return;
    }

    const timeoutId = setTimeout(async () => {
      setIsCheckingCode(true);
      try {
        const result = await checkCodeAction(customCode);
        if (result.success) {
          setCodeAvailable(result.available ?? null);
        }
      } catch (err) {
        console.error("Failed to check code availability:", err);
      } finally {
        setIsCheckingCode(false);
      }
    }, 500);

    return () => clearTimeout(timeoutId);
  }, [customCode]);

  const validateUrl = (url: string): boolean => {
    try {
      new URL(url);
      return true;
    } catch {
      return false;
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError(null);

    // Validation
    if (!destinationUrl.trim()) {
      setError("Please enter a destination URL");
      return;
    }

    if (!validateUrl(destinationUrl)) {
      setError("Please enter a valid URL (including http:// or https://)");
      return;
    }

    if (customCode && customCode.length < 4) {
      setError("Custom code must be at least 4 characters");
      return;
    }

    if (customCode && codeAvailable === false) {
      setError("This custom code is already taken");
      return;
    }

    setIsSubmitting(true);

    try {
      const result = await createURLAction({
        destination_url: destinationUrl,
        custom_code: customCode || undefined,
        notes: notes || undefined,
      });

      if (!result.success || !result.data) {
        throw new Error(result.error || "Failed to create URL");
      }

      // Auto-copy short URL to clipboard
      try {
        await navigator.clipboard.writeText(result.data.short_url);
        toast({
          description: "URL created and copied to clipboard!",
          duration: 4000,
        });
      } catch {
        toast({
          description: "URL created successfully!",
          duration: 4000,
        });
      }

      // Reset form
      setDestinationUrl("");
      setCustomCode("");
      setNotes("");
      setShowAdvanced(false);
      setCodeAvailable(null);

      // Close modal
      onOpenChange(false);

      // Refresh the page
      router.refresh();

      // Call success callback
      if (onSuccess) {
        onSuccess();
      }
    } catch (err: any) {
      console.error("Failed to create URL:", err);
      setError(err.message || "Failed to create URL. Please try again.");
    } finally {
      setIsSubmitting(false);
    }
  };

  // Reset form when modal closes
  useEffect(() => {
    if (!open) {
      setTimeout(() => {
        setDestinationUrl("");
        setCustomCode("");
        setNotes("");
        setError(null);
        setShowAdvanced(false);
        setCodeAvailable(null);
      }, 200);
    }
  }, [open]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>Create Short URL</DialogTitle>
          <DialogDescription>
            Enter a destination URL to create a short link. Customize it with a custom code if you want.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          <div className="space-y-2">
            <Label htmlFor="destination">Destination URL *</Label>
            <Input
              id="destination"
              type="url"
              placeholder="https://example.com/your-long-url"
              value={destinationUrl}
              onChange={(e) => setDestinationUrl(e.target.value)}
              disabled={isSubmitting}
              autoFocus
              required
            />
          </div>

          <div>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => setShowAdvanced(!showAdvanced)}
              className="flex items-center gap-1 text-sm"
            >
              {showAdvanced ? (
                <>
                  <ChevronUp className="h-4 w-4" />
                  Hide advanced options
                </>
              ) : (
                <>
                  <ChevronDown className="h-4 w-4" />
                  Show advanced options
                </>
              )}
            </Button>

            {showAdvanced && (
              <div className="space-y-4 mt-4 p-4 border rounded-lg bg-muted/30">
                <div className="space-y-2">
                  <Label htmlFor="customCode">
                    Custom Code (optional)
                  </Label>
                  <div className="flex items-center gap-2">
                    <Input
                      id="customCode"
                      type="text"
                      placeholder="my-custom-link"
                      value={customCode}
                      onChange={(e) => setCustomCode(e.target.value)}
                      disabled={isSubmitting}
                      pattern="[a-zA-Z0-9-]+"
                      minLength={4}
                      maxLength={12}
                    />
                    {isCheckingCode && (
                      <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
                    )}
                    {!isCheckingCode && codeAvailable === true && (
                      <CheckCircle2 className="h-4 w-4 text-green-500" />
                    )}
                    {!isCheckingCode && codeAvailable === false && (
                      <XCircle className="h-4 w-4 text-red-500" />
                    )}
                  </div>
                  <p className="text-xs text-muted-foreground">
                    4-12 characters (letters, numbers, hyphens)
                  </p>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="notes">Notes (optional)</Label>
                  <Input
                    id="notes"
                    type="text"
                    placeholder="Add a note to remember this link"
                    value={notes}
                    onChange={(e) => setNotes(e.target.value)}
                    disabled={isSubmitting}
                  />
                </div>
              </div>
            )}
          </div>

          <div className="flex justify-end gap-2 pt-4">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isSubmitting}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Creating...
                </>
              ) : (
                "Create"
              )}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}

"use client";

import { Plus } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

interface CreateURLButtonProps {
  onClick: () => void;
  className?: string;
}

export function CreateURLButton({ onClick, className }: CreateURLButtonProps) {
  return (
    <Button
      size="lg"
      className={cn(
        "fixed bottom-24 right-6 h-14 w-14 rounded-full shadow-lg",
        "hover:scale-110 transition-transform duration-200",
        "z-[60]",
        className
      )}
      onClick={onClick}
      aria-label="Create new URL"
    >
      <Plus className="h-6 w-6" />
    </Button>
  );
}

"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useToast } from "@/hooks/use-toast";
import { MoreHorizontal, Edit, Trash2, Power, PowerOff } from "lucide-react";
import { URLResponse } from "@/lib/api";
import { updateURLAction, deleteURLAction } from "@/app/actions";

interface URLActionsProps {
  url: URLResponse;
  onUpdate?: (silent?: boolean) => void;
}

export function URLActions({ url, onUpdate }: URLActionsProps) {
  const router = useRouter();
  const { toast } = useToast();

  const [editDialogOpen, setEditDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const [destinationUrl, setDestinationUrl] = useState(url.destination_url);
  const [notes, setNotes] = useState(url.notes || "");

  const handleEdit = async () => {
    try {
      setIsSubmitting(true);

      const result = await updateURLAction(url.id, {
        destination_url: destinationUrl !== url.destination_url ? destinationUrl : undefined,
        notes: notes !== (url.notes || "") ? notes : undefined,
      });

      if (!result.success) {
        throw new Error(result.error || "Failed to update URL");
      }

      toast({
        description: "URL updated successfully!",
        duration: 3000,
      });

      setEditDialogOpen(false);
      router.refresh();
      if (onUpdate) onUpdate();
    } catch (error: any) {
      toast({
        variant: "destructive",
        description: error.message || "Failed to update URL",
        duration: 3000,
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleDelete = async () => {
    try {
      setIsSubmitting(true);

      const result = await deleteURLAction(url.id);

      if (!result.success) {
        throw new Error(result.error || "Failed to delete URL");
      }

      toast({
        description: "URL deleted successfully!",
        duration: 3000,
      });

      setDeleteDialogOpen(false);
      router.refresh();
      if (onUpdate) onUpdate();
    } catch (error: any) {
      toast({
        variant: "destructive",
        description: error.message || "Failed to delete URL",
        duration: 3000,
      });
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleToggleActive = async () => {
    try {
      const result = await updateURLAction(url.id, {
        is_active: !url.is_active,
      });

      if (!result.success) {
        throw new Error(result.error || "Failed to update URL");
      }

      toast({
        description: url.is_active ? "URL deactivated" : "URL activated",
        duration: 3000,
      });

      // router.refresh();
      if (onUpdate) onUpdate(true);
    } catch (error: any) {
      toast({
        variant: "destructive",
        description: error.message || "Failed to update URL",
        duration: 3000,
      });
    }
  };

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            onClick={(e) => e.stopPropagation()}
          >
            <MoreHorizontal className="h-4 w-4" />
            <span className="sr-only">Open menu</span>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" onClick={(e) => e.stopPropagation()}>
          <DropdownMenuItem
            onClick={(e) => {
              e.stopPropagation();
              setEditDialogOpen(true);
            }}
          >
            <Edit className="mr-2 h-4 w-4" />
            Edit
          </DropdownMenuItem>
          <DropdownMenuItem
            onClick={(e) => {
              e.stopPropagation();
              handleToggleActive();
            }}
          >
            {url.is_active ? (
              <>
                <PowerOff className="mr-2 h-4 w-4" />
                Deactivate
              </>
            ) : (
              <>
                <Power className="mr-2 h-4 w-4" />
                Activate
              </>
            )}
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            onClick={(e) => {
              e.stopPropagation();
              setDeleteDialogOpen(true);
            }}
            className="text-destructive focus:text-destructive"
          >
            <Trash2 className="mr-2 h-4 w-4" />
            Delete
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      {/* Edit Dialog */}
      <Dialog open={editDialogOpen} onOpenChange={setEditDialogOpen}>
        <DialogContent onClick={(e) => e.stopPropagation()}>
          <DialogHeader>
            <DialogTitle>Edit URL</DialogTitle>
            <DialogDescription>
              Update the destination URL or notes for this short link.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="edit-destination">Destination URL</Label>
              <Input
                id="edit-destination"
                type="url"
                value={destinationUrl}
                onChange={(e) => setDestinationUrl(e.target.value)}
                disabled={isSubmitting}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="edit-notes">Notes</Label>
              <Input
                id="edit-notes"
                type="text"
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
                disabled={isSubmitting}
                placeholder="Optional notes"
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setEditDialogOpen(false)}
              disabled={isSubmitting}
            >
              Cancel
            </Button>
            <Button onClick={handleEdit} disabled={isSubmitting}>
              {isSubmitting ? "Saving..." : "Save Changes"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete Confirmation Dialog */}
      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent onClick={(e) => e.stopPropagation()}>
          <DialogHeader>
            <DialogTitle>Delete URL</DialogTitle>
            <DialogDescription>
              Are you sure you want to delete this short URL? This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <div className="py-4">
            <p className="text-sm text-muted-foreground break-all">
              <span className="font-medium">Short URL:</span> {url.short_url}
            </p>
            <p className="text-sm text-muted-foreground break-all mt-2">
              <span className="font-medium">Destination:</span> {url.destination_url}
            </p>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setDeleteDialogOpen(false)}
              disabled={isSubmitting}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={isSubmitting}
            >
              {isSubmitting ? "Deleting..." : "Delete"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

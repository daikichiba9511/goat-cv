import { useRef } from "react";
import { Upload } from "lucide-react";
import type { ImageMeta, ImageStatus } from "../../types";
import { useProjectStore } from "../../stores/projectStore";

type Props = {
  images: ImageMeta[];
  currentImageId: string | null;
  onSelectImage: (img: ImageMeta) => void;
};

export default function Sidebar({ images, currentImageId, onSelectImage }: Props) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const { uploadImage, imageFilters, setImageFilters } = useProjectStore();

  const handleStatusFilter = (value: string) => {
    const nextFilters = { ...imageFilters };
    if (value === "all") {
      delete nextFilters.status;
    } else {
      nextFilters.status = value as ImageStatus;
    }
    void setImageFilters(nextFilters);
  };

  const handleEscalationFilter = (value: string) => {
    const nextFilters = { ...imageFilters };
    if (value === "all") {
      delete nextFilters.escalated;
    } else {
      nextFilters.escalated = value === "true";
    }
    void setImageFilters(nextFilters);
  };

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (!files) return;
    for (const file of Array.from(files)) {
      await uploadImage(file);
    }
    if (fileInputRef.current) {
      fileInputRef.current.value = "";
    }
  };

  return (
    <div className="flex w-28 flex-shrink-0 flex-col border-r bg-white md:w-56">
      <div className="p-3 border-b">
        <button
          type="button"
          onClick={() => fileInputRef.current?.click()}
          aria-label="Upload images"
          title="Upload images"
          className="flex w-full items-center justify-center gap-2 rounded bg-blue-600 py-1.5 text-sm text-white hover:bg-blue-700"
        >
          <Upload aria-hidden="true" size={15} strokeWidth={1.75} />
          <span className="hidden md:inline">Upload Images</span>
        </button>
        <input
          ref={fileInputRef}
          type="file"
          accept="image/*"
          multiple
          onChange={handleUpload}
          className="hidden"
        />
      </div>

      <div className="grid grid-cols-1 gap-1.5 border-b p-2 md:grid-cols-2">
        <select
          aria-label="Filter images by lifecycle"
          value={imageFilters.status ?? "all"}
          onChange={(event) => handleStatusFilter(event.target.value)}
          className="min-w-0 border bg-white px-1.5 py-1 text-xs text-gray-700"
        >
          <option value="all">All states</option>
          <option value="pending">Pending</option>
          <option value="annotated">Annotated</option>
          <option value="in_review">In review</option>
          <option value="rejected">Rejected</option>
          <option value="approved">Approved</option>
        </select>
        <select
          aria-label="Filter images by escalation"
          value={imageFilters.escalated === undefined
            ? "all"
            : String(imageFilters.escalated)}
          onChange={(event) => handleEscalationFilter(event.target.value)}
          className="min-w-0 border bg-white px-1.5 py-1 text-xs text-gray-700"
        >
          <option value="all">All flags</option>
          <option value="false">Clear</option>
          <option value="true">Escalated</option>
        </select>
      </div>

      <div className="flex-1 overflow-y-auto">
        {images.map((img) => (
          <button
            type="button"
            key={img.id}
            onClick={() => onSelectImage(img)}
            className={`block w-full border-b px-3 py-2 text-left ${
              img.id === currentImageId
                ? "bg-blue-50 text-blue-700 font-medium"
                : "hover:bg-gray-50"
            }`}
          >
            <span className="block truncate text-sm">{img.filename}</span>
            <span className="mt-0.5 flex items-center gap-1 text-xs font-normal text-gray-500">
              <span className="truncate">{img.status.replace("_", " ")}</span>
              {img.escalated && <span className="text-orange-700">/ escalated</span>}
            </span>
          </button>
        ))}
      </div>
    </div>
  );
}

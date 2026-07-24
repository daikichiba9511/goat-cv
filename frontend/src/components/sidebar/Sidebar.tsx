import { useRef } from "react";
import { Upload } from "lucide-react";
import type { ImageMeta } from "../../types";
import { useProjectStore } from "../../stores/projectStore";

type Props = {
  images: ImageMeta[];
  currentImageId: string | null;
  onSelectImage: (img: ImageMeta) => void;
};

export default function Sidebar({ images, currentImageId, onSelectImage }: Props) {
  const fileInputRef = useRef<HTMLInputElement>(null);
  const { uploadImage } = useProjectStore();

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

      <div className="flex-1 overflow-y-auto">
        {images.map((img) => (
          <div
            key={img.id}
            onClick={() => onSelectImage(img)}
            className={`px-3 py-2 cursor-pointer text-sm truncate border-b ${
              img.id === currentImageId
                ? "bg-blue-50 text-blue-700 font-medium"
                : "hover:bg-gray-50"
            }`}
          >
            {img.filename}
          </div>
        ))}
      </div>
    </div>
  );
}

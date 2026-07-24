import ReactMarkdown, { type Components } from "react-markdown";
import remarkGfm from "remark-gfm";

type Props = {
  body: string;
  components: Components;
};

// SafeMarkdown renders user-authored Markdown without raw HTML or remote images.
export default function SafeMarkdown({ body, components }: Props) {
  return (
    // Why: Markdownの利用箇所で安全化設定がずれないよう、実行可能HTMLと外部画像の除外を共通化する。
    <ReactMarkdown
      skipHtml
      remarkPlugins={[remarkGfm]}
      disallowedElements={["img"]}
      components={components}
    >
      {body}
    </ReactMarkdown>
  );
}

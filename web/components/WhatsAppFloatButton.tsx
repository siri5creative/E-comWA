// PRD 6.1 / IMPLEMENTATION.md file structure. No interactivity needed, so
// this stays a Server Component. Renders nothing if the store hasn't
// configured a WhatsApp number yet.
export function WhatsAppFloatButton() {
  const number = process.env.NEXT_PUBLIC_WHATSAPP_NUMBER;
  if (!number) {
    return null;
  }

  return (
    <a
      href={`https://wa.me/${number}`}
      target="_blank"
      rel="noopener noreferrer"
      aria-label="Chat via WhatsApp"
      className="fixed bottom-6 right-6 z-50 flex h-14 w-14 items-center justify-center rounded-full bg-[#25D366] text-white shadow-lg transition-transform hover:scale-105"
    >
      <svg viewBox="0 0 32 32" width="28" height="28" fill="currentColor" aria-hidden="true">
        <path d="M16.004 3C9.377 3 4 8.373 4 15c0 2.34.688 4.523 1.875 6.36L4 29l7.836-1.84A11.94 11.94 0 0 0 16.004 27C22.63 27 28 21.627 28 15S22.63 3 16.004 3Zm0 21.818a9.77 9.77 0 0 1-4.98-1.362l-.357-.213-4.65 1.092 1.107-4.53-.234-.372A9.78 9.78 0 0 1 5.2 15c0-5.964 4.84-10.804 10.804-10.804S26.808 9.036 26.808 15 21.968 24.818 16.004 24.818Zm5.63-7.516c-.308-.154-1.82-.898-2.103-1-.283-.103-.489-.154-.694.154-.206.308-.797 1-.977 1.206-.18.206-.36.231-.668.077-.308-.154-1.3-.479-2.475-1.526-.915-.816-1.532-1.824-1.712-2.132-.18-.308-.02-.474.135-.628.138-.137.308-.36.463-.54.154-.18.206-.308.309-.514.103-.206.051-.386-.026-.54-.077-.154-.694-1.673-.951-2.29-.25-.6-.505-.52-.694-.53-.18-.008-.386-.01-.591-.01-.206 0-.54.077-.823.386-.283.308-1.08 1.056-1.08 2.575 0 1.52 1.106 2.987 1.26 3.193.154.206 2.176 3.322 5.272 4.658.737.318 1.311.508 1.759.65.739.235 1.412.202 1.944.123.593-.089 1.82-.744 2.077-1.463.257-.72.257-1.336.18-1.464-.077-.128-.283-.206-.591-.36Z" />
      </svg>
    </a>
  );
}

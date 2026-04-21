export type ApiParameter = {
  name: string;
  type: string;
  required: boolean;
  description: string;
  example?: string;
};

export type ApiEndpoint = {
  id: string;
  method: "GET" | "POST";
  path: string;
  title: string;
  description: string;
  parameters?: ApiParameter[];
  requestBody?: {
    description: string;
    example: Record<string, unknown>;
  };
};

export const apiEndpoints: ApiEndpoint[] = [
  {
    id: "health",
    method: "GET",
    path: "/health",
    title: "Health Check",
    description: "Check whether API server is healthy.",
  },
  {
    id: "search",
    method: "GET",
    path: "/api/search",
    title: "Search Tracks",
    description: "Search tracks from Spotify or Deezer.",
    parameters: [
      { name: "q", type: "string", required: true, description: "Search query", example: "wildflower" },
      { name: "source", type: "string", required: false, description: "spotify or deezer", example: "spotify" },
    ],
  },
  {
    id: "metadata",
    method: "GET",
    path: "/api/metadata",
    title: "Get Metadata",
    description: "Resolve metadata from Spotify/Deezer URL.",
    parameters: [
      {
        name: "url",
        type: "string",
        required: true,
        description: "Track/album/playlist URL",
        example: "https://open.spotify.com/track/4iV5W9uYEdYUVa79Axb7Rh",
      },
    ],
  },
  {
    id: "parse-url",
    method: "GET",
    path: "/api/parse-url",
    title: "Parse URL",
    description: "Extract item type and ID from URL.",
    parameters: [
      {
        name: "url",
        type: "string",
        required: true,
        description: "Spotify/Deezer URL",
      },
    ],
  },
  {
    id: "availability",
    method: "GET",
    path: "/api/availability",
    title: "Check Availability",
    description: "Check provider availability by spotify/deezer/isrc IDs.",
  },
  {
    id: "download",
    method: "POST",
    path: "/api/download",
    title: "Download Track",
    description: "Start track download job.",
  },
  {
    id: "progress",
    method: "GET",
    path: "/api/progress",
    title: "Get Progress",
    description: "Read progress for all or one download job.",
  },
  {
    id: "files",
    method: "GET",
    path: "/api/files/{filename}",
    title: "Download File",
    description: "Fetch completed file by filename.",
  },
  {
    id: "lyrics",
    method: "GET",
    path: "/api/lyrics",
    title: "Fetch Lyrics",
    description: "Fetch lyrics by track and artist metadata.",
  },
];

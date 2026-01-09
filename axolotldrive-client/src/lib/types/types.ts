export type FileAndFolder = {
    id: string;
    name: string;
    type: "file" | "folder";
    size: number; // in megabytes
    extension?: string; // only for files
    createdAt: Date;
    updatedAt: Date;
}
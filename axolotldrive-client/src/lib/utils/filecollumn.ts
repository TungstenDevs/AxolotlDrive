import type { FileAndFolder } from "$lib/types/types";
// import type { ColumnDef } from "@tanstack/table-core";

export const fileTableData: FileAndFolder[] = [
	{
		id: "1",
		name: "Document.pdf",
		type: "file",
		size: 2.5,
		createdAt: new Date("2024-01-01T10:00:00Z"),
		updatedAt: new Date("2024-01-02T12:00:00Z"),
		selected: false,
	},
	{
		id: "2",
		name: "Photos",
		type: "folder",
		size: 0,
		createdAt: new Date("2024-02-15T09:30:00Z"),
		updatedAt: new Date("2024-02-16T11:45:00Z"),
		selected: false,
	},
	{
		id: "3",
		name: "Presentation.pptx",
		type: "file",
		size: 5.0,
		createdAt: new Date("2024-03-10T14:20:00Z"),
		updatedAt: new Date("2024-03-11T16:25:00Z"),
		selected: false,
	},
	{
		id: "4",
		name: "Music",
		type: "folder",
		size: 0,
		createdAt: new Date("2024-04-05T08:15:00Z"),
		updatedAt: new Date("2024-04-06T10:10:00Z"),
		selected: false,
	},
	{
		id: "5",
		name: "Spreadsheet.xlsx",
		type: "file",
		size: 3.2,
		createdAt: new Date("2024-05-20T13:50:00Z"),
		updatedAt: new Date("2024-05-21T15:55:00Z"),
		selected: false,
	},
	{
		id: "6",
		name: "Videos",
		type: "folder",
		size: 0,
		createdAt: new Date("2024-06-01T11:00:00Z"),
		updatedAt: new Date("2024-06-02T13:00:00Z"),
		selected: false,
	},
	{
		id: "7",
		name: "Report.docx",
		type: "file",
		size: 1.8,
		createdAt: new Date("2024-07-10T09:00:00Z"),
		updatedAt: new Date("2024-07-11T11:00:00Z"),
		selected: false,
	},
	{
		id: "8",
		name: "Music",
		type: "folder",
		size: 0,
		createdAt: new Date("2024-08-15T14:30:00Z"),
		updatedAt: new Date("2024-08-16T16:30:00Z"),
		selected: false,
	},
	{
		id: "9",
		name: "Diagram.svg",
		type: "file",
		size: 0.9,
		createdAt: new Date("2024-09-05T12:20:00Z"),
		updatedAt: new Date("2024-09-06T14:25:00Z"),
		selected: false,
	},
	{
		id: "10",
		name: "Archives",
		type: "folder",
		size: 0,
		createdAt: new Date("2024-10-01T10:10:00Z"),
		updatedAt: new Date("2024-10-02T12:15:00Z"),
		selected: false,
	},
	{
		id: "11",
		name: "Code.js",
		type: "file",
		size: 0.5,
		createdAt: new Date("2024-11-11T08:00:00Z"),
		updatedAt: new Date("2024-11-12T09:00:00Z"),
		selected: false,
	},
	{
		id: "12",
		name: "Backups",
		type: "folder",
		size: 0,
		createdAt: new Date("2024-12-01T16:00:00Z"),
		updatedAt: new Date("2024-12-02T17:00:00Z"),
		selected: false,
	},
];

// export const fileTableColumns: ColumnDef<FileAndFolder>[] = [
// 	{
// 		id: "select",
// 		header: ({ table }) =>
// 			renderComponent(Checkbox, {
// 				checked: table.getIsAllPageRowsSelected(),
// 				indeterminate:
// 					table.getIsSomePageRowsSelected() &&
// 					!table.getIsAllPageRowsSelected(),
// 				onCheckedChange: (value) => table.toggleAllPageRowsSelected(!!value),
// 				"aria-label": "Select all",
// 			}),
// 		cell: ({ row }) =>
// 			renderComponent(Checkbox, {
// 				checked: row.getIsSelected(),
// 				onCheckedChange: (value) => row.toggleSelected(!!value),
// 				"aria-label": "Select row",
// 			}),
// 		enableSorting: false,
// 		enableHiding: false,
// 	},
// 	{
// 		accessorKey: "name",
// 		header: "Name",
// 	},
// 	{
// 		accessorKey: "type",
// 		header: "Type",
// 	},
// 	{
// 		accessorKey: "size",
// 		header: "Size (MB)",
// 	},
// 	{
// 		accessorKey: "createdAt",
// 		header: "Created",
// 		cell: (info) =>
// 			new Date(info.getValue() as Date).toLocaleDateString(),
// 	},
// 	{
// 		accessorKey: "updatedAt",
// 		header: "Updated",
// 		cell: (info) =>
// 			new Date(info.getValue() as Date).toLocaleDateString(),
// 	},
// ];

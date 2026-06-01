export default function MaintenanceLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold text-gray-900 dark:text-white">
          Maintenance Tasks
        </h1>
        <p className="text-gray-600 dark:text-gray-400">
          Track and manage property maintenance tasks
        </p>
      </div>
      {children}
    </div>
  );
}

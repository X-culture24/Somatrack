#!/bin/bash

# Installation script for St. Mary's LMS UI Component Integration
# This script installs all required dependencies for shadcn-style components

echo "🎨 St. Mary's LMS - UI Component Integration Setup"
echo "=================================================="
echo ""

# Check if we're in the frontend directory
if [ ! -f "package.json" ]; then
    echo "❌ Error: package.json not found"
    echo "Please run this script from the /frontend directory"
    exit 1
fi

echo "📦 Installing required npm packages..."
echo ""

# Install Radix UI primitives
npm install @radix-ui/react-slot @radix-ui/react-tabs @radix-ui/react-label

# Install class-variance-authority for button variants
npm install class-variance-authority

echo ""
echo "✅ All dependencies installed successfully!"
echo ""
echo "📁 Created files:"
echo "  ✓ src/lib/utils.ts"
echo "  ✓ src/components/ui/card.tsx"
echo "  ✓ src/components/ui/button.tsx"
echo "  ✓ src/components/ui/tabs.tsx"
echo "  ✓ src/components/ui/input.tsx"
echo "  ✓ src/components/ui/label.tsx"
echo "  ✓ src/components/ui/border-beam.tsx"
echo "  ✓ src/features/admin/AdminHomeEnhanced.tsx"
echo ""
echo "⚙️  Updated files:"
echo "  ✓ src/index.css (added theme + animations)"
echo "  ✓ tsconfig.app.json (added path aliases)"
echo "  ✓ vite.config.ts (added path resolution)"
echo ""
echo "🎯 Next steps:"
echo ""
echo "1. Review the INTEGRATION_GUIDE.md for detailed documentation"
echo "2. Start the dev server: npm run dev"
echo "3. Test the enhanced dashboard at /admin"
echo "4. Apply the same patterns to other dashboards:"
echo "   - TeacherHome.tsx"
echo "   - StudentHome.tsx"
echo "   - ParentHome.tsx"
echo "   - FinanceHome.tsx"
echo ""
echo "📚 Component Usage Examples:"
echo ""
echo "Import shadcn components:"
echo "  import { Card, CardHeader, CardTitle, CardContent } from '@/components/ui/card'"
echo "  import { Button } from '@/components/ui/button'"
echo "  import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'"
echo "  import { BorderBeam } from '@/components/ui/border-beam'"
echo ""
echo "Use existing school components:"
echo "  import { kes, Badge, Stat } from '@/components/ui'"
echo ""
echo "✨ Happy coding!"

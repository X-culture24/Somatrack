# Frontend Integration Guide - Mock-up to St. Mary's LMS

## Overview
This guide helps integrate the modern UI components from the mock-up into the existing St. Mary's LMS frontend.

## Setup Steps

### 1. Install Required NPM Dependencies

Run this command in the `/frontend` directory:

```bash
npm install @radix-ui/react-slot @radix-ui/react-tabs @radix-ui/react-label class-variance-authority
```

### 2. Project Structure
The project follows a shadcn-compatible structure:

```
frontend/src/
├── lib/
│   └── utils.ts          ✅ Created (cn utility)
├── components/
│   └── ui/               ✅ Ready for shadcn components
│       ├── index.tsx     ✅ Existing (school-themed components)
│       ├── card.tsx      ✅ Created (shadcn Card)
│       ├── button.tsx    ✅ Created (shadcn Button)
│       ├── tabs.tsx      ✅ Created (shadcn Tabs)
│       ├── input.tsx     ✅ Created (shadcn Input)
│       ├── label.tsx     ✅ Created (shadcn Label)
│       └── border-beam.tsx ✅ Created (magic-ui BorderBeam)
├── features/
│   ├── admin/            🎨 Dashboards to enhance
│   ├── teacher/          🎨 Dashboards to enhance
│   ├── student/          🎨 Dashboards to enhance
│   ├── parent/           🎨 Dashboards to enhance
│   └── finance/          🎨 Dashboards to enhance
└── index.css             ✅ Updated with full theme + animations
```

### 3. Theme Configuration

The CSS has been updated with:
- ✅ Full shadcn theme variables (light + dark mode)
- ✅ School brand colors (Navy #1b365d, Light Blue #4a9eca)
- ✅ All keyframe animations (border-beam, accordion, fade, etc.)
- ✅ Breakpoints and radius tokens
- ✅ Gradient and sidebar colors

### 4. Path Aliases

Ensure your `tsconfig.json` has the path alias configured:

```json
{
  "compilerOptions": {
    "paths": {
      "@/*": ["./src/*"]
    }
  }
}
```

### 5. Available Components

#### Shadcn Components (New)
- `Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter` - from `@/components/ui/card`
- `Button` - from `@/components/ui/button` (with variants: default, destructive, outline, secondary, ghost, link)
- `Tabs, TabsList, TabsTrigger, TabsContent` - from `@/components/ui/tabs`
- `Input` - from `@/components/ui/input`
- `Label` - from `@/components/ui/label`
- `BorderBeam` - from `@/components/ui/border-beam`

#### School Components (Existing)
- All existing components in `@/components/ui/index` remain intact
- `Card, Button, Input, Select, Field, Stat, PageHeader, Table, Badge, kes()`
- `ConfirmDialog, useConfirm`

### 6. Usage Examples

#### Using New Card Component
```tsx
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"

export function DashboardCard() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Students</CardTitle>
      </CardHeader>
      <CardContent>
        <p className="text-3xl font-bold text-[#1b365d]">1,245</p>
      </CardContent>
    </Card>
  )
}
```

#### Using BorderBeam for Animated Cards
```tsx
import { Card } from "@/components/ui/card"
import { BorderBeam } from "@/components/ui/border-beam"

export function FeaturedCard() {
  return (
    <Card className="relative">
      <BorderBeam 
        size={250}
        duration={12}
        colorFrom="#4a9eca"
        colorTo="#1b365d"
      />
      <CardContent>
        <h3>Premium Feature</h3>
      </CardContent>
    </Card>
  )
}
```

#### Using Tabs
```tsx
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"

export function ReportTabs() {
  return (
    <Tabs defaultValue="overview">
      <TabsList>
        <TabsTrigger value="overview">Overview</TabsTrigger>
        <TabsTrigger value="details">Details</TabsTrigger>
      </TabsList>
      <TabsContent value="overview">
        Overview content...
      </TabsContent>
      <TabsContent value="details">
        Detail content...
      </TabsContent>
    </Tabs>
  )
}
```

## Integration with Mock-up Features

### Mock-up Screens to Integrate

Based on the mock-up images, here are the features to integrate:

1. **Enhanced Dashboard Cards** - Use BorderBeam for featured metrics
2. **Tabbed Reports** - Replace simple layouts with Tabs component
3. **Modern Forms** - Use Label + Input combinations
4. **Stat Cards with Animation** - Animated stat displays
5. **Action Buttons** - Updated button variants

### Dashboard Enhancement Strategy

For each dashboard (`AdminHome.tsx`, `TeacherHome.tsx`, etc.):

1. **Keep existing content and data fetching** - Don't change business logic
2. **Enhance with new UI components** - Wrap in new Card components
3. **Add animations** - Use BorderBeam for important metrics
4. **Improve layouts** - Use Grid layouts from mock-up
5. **Add tabs where appropriate** - For multi-view dashboards

### Color Scheme Consistency

All components automatically use the school theme:
- **Primary (Navy)**: `#1b365d` - Buttons, headings, emphasis
- **Accent (Light Blue)**: `#4a9eca` - Interactive elements, links, badges
- **Background**: `#f0f4f8` - Page background
- **Cards**: White with subtle shadows

## Next Steps

1. ✅ Install dependencies: `npm install @radix-ui/react-slot @radix-ui/react-tabs @radix-ui/react-label class-variance-authority`
2. ✅ Verify path aliases in tsconfig.json
3. 🎨 Start enhancing dashboards with new components
4. 🎨 Add BorderBeam to featured metrics
5. 🎨 Convert layouts to use Tabs where appropriate
6. ✅ Test in dev mode: `npm run dev`

## Troubleshooting

### Import Errors
If you see module not found errors for `@/lib/utils` or `@/components/ui/*`:
- Check that `tsconfig.json` has the `@/*` path alias
- Restart your dev server after adding new files

### Styling Issues
- Make sure Tailwind 4 is configured correctly
- Check that `@import "tailwindcss"` is at the top of index.css
- Clear browser cache if styles don't update

### Animation Not Working
- Verify all keyframe animations are in index.css
- Check that the component using animations has proper CSS variable support

## Support

For questions or issues:
1. Check existing component implementations in `/src/components/ui/index.tsx`
2. Review shadcn documentation: https://ui.shadcn.com
3. Check Tailwind CSS v4 docs: https://tailwindcss.com

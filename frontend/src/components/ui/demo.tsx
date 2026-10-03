import { useNavigate } from "react-router-dom";
import { Hero195 } from "@/components/ui/hero-195";

const DemoOne = () => {
  const navigate = useNavigate();
  return (
    <div className="min-h-screen">
      <Hero195
        primaryAction={{
          label: "Sign in to portal",
          onClick: () => navigate("/login"),
        }}
        secondaryAction={{
          label: "Parent short link",
          onClick: () => navigate("/r"),
        }}
      />
    </div>
  );
};

export { DemoOne };

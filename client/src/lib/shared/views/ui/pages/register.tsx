import Register from "@/lib/auth/views/ui/register"
import ClearLayout from "@/lib/shared/views/ui/layouts/clear"
import useMeta from "@/lib/shared/views/hooks/use-meta"

function RegisterPage() {
    useMeta("Jota | Register", "Create your account on Jota")
    return (
        <ClearLayout className="gap-4">
            <Register />
        </ClearLayout>
    )
}

export default RegisterPage
package main

import "math"

var gaussian = [3]float64{-math.Sqrt(0.6), 0, math.Sqrt(0.6)}

var localPoints2D = [8][2]float64{
	{-1, -1}, {1, -1}, {1, 1}, {-1, 1},
	{0, -1}, {1, 0}, {0, 1}, {-1, 0},
}

var localPoints3D = [20][3]float64{
	{-1, 1, -1}, {1, 1, -1}, {1, -1, -1}, {-1, -1, -1},
	{-1, 1, 1}, {1, 1, 1}, {1, -1, 1}, {-1, -1, 1},
	{0, 1, -1}, {1, 0, -1}, {0, -1, -1}, {-1, 0, -1},
	{-1, 1, 0}, {1, 1, 0}, {1, -1, 0}, {-1, -1, 0},
	{0, 1, 1}, {1, 0, 1}, {0, -1, 1}, {-1, 0, 1},
}

var dfiabg [3 * 3 * 3][20][3]float64

var depsite [3 * 3][8][2]float64

var depsiXYZdeNT [3 * 3][8]float64

var gaussianCoefficients = [3]float64{5.0 / 9.0, 8.0 / 9.0, 5.0 / 9.0}

func init() {
	calculateDFIABG()
	calculateDEPSITE()
	calculateDepsiXYZdeNT()
}

func calculateDFIABG() {
	for k1, gamma := range gaussian {
		for k2, beta := range gaussian {
			for k3, alpha := range gaussian {
				for i, point := range localPoints3D {
					if i <= 7 {
						dfiabg[k1*9+k2*3+k3][i] = dfiabg18(alpha, beta, gamma, point[0], point[1], point[2])
					} else {
						dfiabg[k1*9+k2*3+k3][i] = dfiabg14(alpha, beta, gamma, point[0], point[1], point[2])
					}
				}
			}
		}
	}
}

func dfiabg18(alpha, beta, gamma, x, y, z float64) [3]float64 {
	return [3]float64{
		(1.0 / 8.0) * (1 + beta*y) * (1 + gamma*z) * (x*(-2+alpha*x+gamma*z+beta*y) + x*(1+alpha*x)),
		(1.0 / 8.0) * (1 + alpha*x) * (1 + gamma*z) * (y*(-2+alpha*x+gamma*z+beta*y) + y*(1+beta*y)),
		(1.0 / 8.0) * (1 + beta*y) * (1 + alpha*x) * (z*(-2+alpha*x+gamma*z+beta*y) + z*(1+gamma*z)),
	}
}

func dfiabg14(alpha, beta, gamma float64, alphaI, betaI, gammaI float64) [3]float64 {
	return [3]float64{
		(1.0 / 4.0) * (1 + beta*betaI) * (1 + gamma*gammaI) *
			(alphaI*(-betaI*betaI*gammaI*gammaI*alpha*alpha-beta*beta*gammaI*gammaI*alphaI*alphaI-
				betaI*betaI*gamma*gamma*alphaI*alphaI+1) - (2*betaI*betaI*gammaI*gammaI*alpha)*(alpha*alphaI+1)),

		(1.0 / 4.0) * (1 + alpha*alphaI) * (1 + gamma*gammaI) *
			(betaI*(-betaI*betaI*gammaI*gammaI*alpha*alpha-beta*beta*gammaI*gammaI*alphaI*alphaI-
				betaI*betaI*gamma*gamma*alphaI*alphaI+1) - (2*beta*gammaI*gammaI*alphaI*alphaI)*(betaI*beta+1)),

		(1.0 / 4.0) * (1 + beta*betaI) * (1 + alpha*alphaI) *
			(gammaI*(-betaI*betaI*gammaI*gammaI*alpha*alpha-beta*beta*gammaI*gammaI*alphaI*alphaI-
				betaI*betaI*gamma*gamma*alphaI*alphaI+1) - (2*betaI*betaI*gamma*alphaI*alphaI)*(gamma*gammaI+1)),
	}
}

func calculateDEPSITE() {
	for k1, eta := range gaussian {
		for k2, tau := range gaussian {
			for i, point := range localPoints2D {
				if i < 4 {
					depsite[k1*3+k2][i] = psint14der(eta, tau, point[0], point[1])
				} else if i == 4 || i == 6 {
					depsite[k1*3+k2][i] = psint57der(eta, tau, point[0], point[1])
				} else if i == 5 || i == 7 {
					depsite[k1*3+k2][i] = psint68der(eta, tau, point[0], point[1])
				}
			}
		}
	}
}

func psint14der(eta, tau, x, y float64) [2]float64 {
	return [2]float64{
		(1.0 / 4.0) * (tau*y + 1) * (x*(x*eta+y*tau-1) + x*(x*eta+1)),
		(1.0 / 4.0) * (x*eta + 1) * (y*(x*eta+y*tau-1) + y*(y*tau+1)),
	}
}

func psint57der(eta, tau, _, y float64) [2]float64 {
	return [2]float64{
		(-tau*y - 1) * eta,
		(1.0 / 2.0) * (1 - eta*eta) * y,
	}
}

func psint68der(eta, tau, x, _ float64) [2]float64 {
	return [2]float64{
		(1.0 / 2.0) * (1 - tau*tau) * x,
		(-eta*x - 1) * tau,
	}
}

func calculateDepsiXYZdeNT() {
	for k1, eta := range gaussian {
		for k2, tau := range gaussian {
			for i, point := range localPoints2D {
				if i < 4 {
					depsiXYZdeNT[k1*3+k2][i] = psint14(eta, tau, point[0], point[1])
				} else if i == 4 || i == 6 {
					depsiXYZdeNT[k1*3+k2][i] = psint57(eta, tau, point[0], point[1])
				} else if i == 5 || i == 7 {
					depsiXYZdeNT[k1*3+k2][i] = psint68(eta, tau, point[0], point[1])
				}
			}
		}
	}
}

func psint14(eta, tau, x, y float64) float64 {
	return (1.0 / 4.0) * (tau*y + 1) * (eta*x + 1) * (eta*x + y*tau - 1)
}

func psint57(eta, tau, _, y float64) float64 {
	return (1.0 / 2.0) * (-eta*eta + 1) * (y*tau + 1)
}

func psint68(eta, tau, x, _ float64) float64 {
	return (1.0 / 2.0) * (-tau*tau + 1) * (x*eta + 1)
}
